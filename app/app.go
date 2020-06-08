package app

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/canonical/go-dqlite"
	"github.com/canonical/go-dqlite/client"
	"github.com/canonical/go-dqlite/driver"
	"github.com/pkg/errors"
)

// App is a high-level helper for initializing a typical dqlite-based Go
// application.
//
// It takes care of starting a dqlite node and registering a dqlite Go SQL
// driver.
type App struct {
	id              uint64
	address         string
	node            *dqlite.Node
	nodeBindAddress string
	listener        net.Listener
	tls             *tlsSetup
	store           client.NodeStore
	driver          *driver.Driver
	driverName      string
	log             client.LogFunc
	stop            context.CancelFunc // Signal App.run() to stop.
	proxyCh         chan struct{}      // Waits for App.proxy() to return.
	runCh           chan struct{}      // Waits for App.run() to return.
	readyCh         chan struct{}      // Waits for startup tasks
	voters          int
}

// New creates a new application node.
func New(dir string, options ...Option) (app *App, err error) {
	o := defaultOptions()
	for _, option := range options {
		option(o)
	}

	// List of cleanup functions to run in case of errors.
	cleanups := []func(){}
	defer func() {
		if err == nil {
			return
		}
		for i := range cleanups {
			i = len(cleanups) - 1 - i // Reverse order
			cleanups[i]()
		}
	}()

	// Load our ID, or generate one if we are joining.
	info := client.NodeInfo{}
	infoFileExists, err := fileExists(dir, infoFile)
	if err != nil {
		return nil, err
	}
	if !infoFileExists {
		if len(o.Cluster) == 0 {
			info.ID = dqlite.BootstrapID
		} else {
			info.ID = dqlite.GenerateID(o.Address)
			if err := fileWrite(dir, joinFile, []byte{}); err != nil {
				return nil, err
			}
		}
		info.Address = o.Address

		if err := fileMarshal(dir, infoFile, info); err != nil {
			return nil, err
		}

		cleanups = append(cleanups, func() { fileRemove(dir, infoFile) })
	} else {
		if err := fileUnmarshal(dir, infoFile, &info); err != nil {
			return nil, err
		}
		if o.Address != "" && o.Address != info.Address {
			return nil, fmt.Errorf("address %q in info.yaml does not match %q", info.Address, o.Address)
		}
	}

	joinFileExists, err := fileExists(dir, joinFile)
	if err != nil {
		return nil, err
	}

	if info.ID == dqlite.BootstrapID && joinFileExists {
		return nil, fmt.Errorf("bootstrap node can't join a cluster")
	}

	// Open the nodes store.
	storeFileExists, err := fileExists(dir, storeFile)
	if err != nil {
		return nil, err
	}
	store, err := client.NewYamlNodeStore(filepath.Join(dir, storeFile))
	if err != nil {
		return nil, fmt.Errorf("open cluster.yaml node store: %w", err)
	}

	// The info file and the store file should both exists or none of them
	// exist.
	if infoFileExists != storeFileExists {
		return nil, fmt.Errorf("inconsistent info.yaml and cluster.yaml")
	}

	if !storeFileExists {
		// If this is a brand new application node, populate the store
		// either with the node's address (for bootstrap nodes) or with
		// the given cluster addresses (for joining nodes).
		nodes := []client.NodeInfo{}
		if info.ID == dqlite.BootstrapID {
			nodes = append(nodes, client.NodeInfo{Address: o.Address})
		} else {
			if len(o.Cluster) == 0 {
				return nil, fmt.Errorf("no cluster addresses provided")
			}
			for _, address := range o.Cluster {
				nodes = append(nodes, client.NodeInfo{Address: address})
			}
		}
		if err := store.Set(context.Background(), nodes); err != nil {
			return nil, fmt.Errorf("initialize node store: %w", err)
		}
		cleanups = append(cleanups, func() { fileRemove(dir, storeFile) })
	}

	// Start the local dqlite engine.
	var nodeBindAddress string
	var nodeDial client.DialFunc
	if o.TLS != nil {
		nodeBindAddress = fmt.Sprintf("@dqlite-%d", info.ID)

		// Within a snap we need to choose a different name for the abstract unix domain
		// socket to get it past the AppArmor confinement.
		// See https://github.com/snapcore/snapd/blob/master/interfaces/apparmor/template.go#L357
		snapInstanceName := os.Getenv("SNAP_INSTANCE_NAME")
		if len(snapInstanceName) > 0 {
			nodeBindAddress = fmt.Sprintf("@snap.%s.dqlite-%d", snapInstanceName, info.ID)
		}

		nodeDial = makeNodeDialFunc(o.TLS.Dial)
	} else {
		nodeBindAddress = o.Address
		nodeDial = client.DefaultDialFunc
	}
	node, err := dqlite.New(
		info.ID, o.Address, dir,
		dqlite.WithBindAddress(nodeBindAddress),
		dqlite.WithDialFunc(nodeDial),
	)
	if err != nil {
		return nil, fmt.Errorf("create node: %w", err)
	}
	if err := node.Start(); err != nil {
		return nil, fmt.Errorf("start node: %w", err)
	}
	cleanups = append(cleanups, func() { node.Close() })

	// Register the local dqlite driver.
	driverDial := client.DefaultDialFunc
	if o.TLS != nil {
		driverDial = client.DialFuncWithTLS(driverDial, o.TLS.Dial)
	}

	driver, err := driver.New(store, driver.WithDialFunc(driverDial), driver.WithLogFunc(o.Log))
	if err != nil {
		return nil, fmt.Errorf("create driver: %w", err)
	}
	driverIndex++
	driverName := fmt.Sprintf("dqlite-%d", driverIndex)
	sql.Register(driverName, driver)

	ctx, stop := context.WithCancel(context.Background())

	app = &App{
		id:              info.ID,
		address:         o.Address,
		node:            node,
		nodeBindAddress: nodeBindAddress,
		store:           store,
		driver:          driver,
		driverName:      driverName,
		log:             o.Log,
		tls:             o.TLS,
		stop:            stop,
		runCh:           make(chan struct{}, 0),
		readyCh:         make(chan struct{}, 0),
		voters:          o.Voters,
	}

	// Start the proxy if a TLS configuration was provided.
	if o.TLS != nil {
		listener, err := net.Listen("tcp", o.Address)
		if err != nil {
			return nil, fmt.Errorf("listen to %s: %w", o.Address, err)
		}
		proxyCh := make(chan struct{}, 0)

		app.listener = listener
		app.proxyCh = proxyCh

		go app.proxy()

		cleanups = append(cleanups, func() { listener.Close(); <-proxyCh })

	}

	go app.run(ctx, joinFileExists)

	return app, nil
}

// Close the application node, releasing all resources it created.
func (a *App) Close() error {
	// Stop the run goroutine.
	a.stop()
	<-a.runCh

	// Try to transfer leadership if we are the leader.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cli, err := client.New(ctx, a.nodeBindAddress)
	if err == nil {
		leader, err := cli.Leader(ctx)
		if err == nil && leader != nil && leader.Address == a.address {
			cli.Transfer(ctx, 0)
		}
		cli.Close()
	}

	if a.listener != nil {
		a.listener.Close()
		<-a.proxyCh
	}
	if err := a.node.Close(); err != nil {
		return err
	}
	return nil
}

// ID returns the dqlite ID of this application node.
func (a *App) ID() uint64 {
	return a.id
}

// Address returns the dqlite address of this application node.
func (a *App) Address() string {
	return a.address
}

// Driver returns the name used to register the dqlite driver.
func (a *App) Driver() string {
	return a.driverName
}

// Ready can be used to wait for a node to complete some initial tasks that are
// initiated at startup. For example a brand new node will attempt to join the
// cluster, a restarted node will check if it should assume some particular
// role, etc.
//
// If this method returns without error it means that those initial tasks have
// succeeded and follow-up operations like Open() are more likely to succeeed
// quickly.
func (a *App) Ready(ctx context.Context) error {
	select {
	case <-a.readyCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Open the dqlite database with the given name
func (a *App) Open(ctx context.Context, database string) (*sql.DB, error) {
	db, err := sql.Open(a.Driver(), database)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 60; i++ {
		err = db.PingContext(ctx)
		if err == nil {
			break
		}
		cause := errors.Cause(err)
		if cause != driver.ErrNoAvailableLeader {
			return nil, err
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Leader returns a client connected to the current cluster leader, if any.
func (a *App) Leader(ctx context.Context) (*client.Client, error) {
	dial := client.DefaultDialFunc
	if a.tls != nil {
		dial = client.DialFuncWithTLS(dial, a.tls.Dial)
	}
	return client.FindLeader(ctx, a.store, client.WithDialFunc(dial), client.WithLogFunc(a.log))
}

// Proxy incoming TLS connections.
func (a *App) proxy() {
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	for {
		client, err := a.listener.Accept()
		if err != nil {
			cancel()
			wg.Wait()
			close(a.proxyCh)
			return
		}
		address := client.RemoteAddr()
		a.debug("new connection from %s", address)
		server, err := net.Dial("unix", a.nodeBindAddress)
		if err != nil {
			a.error("dial local node: %v", err)
			client.Close()
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proxy(ctx, client, server, a.tls.Listen); err != nil {
				a.error("proxy: %v", err)
			}
		}()
	}
}

// Run background tasks. The join flag is true if the node is a brand new one
// and should join the cluster.
func (a *App) run(ctx context.Context, join bool) {
	defer close(a.runCh)

	delay := time.Duration(0)
	ready := false
	for {
		select {
		case <-ctx.Done():
			// If we didn't become ready yet, close the ready
			// channel, to unblock any call to Ready().
			if !ready {
				close(a.readyCh)
			}
			return
		case <-time.After(delay):
			cli, err := a.Leader(ctx)
			if err != nil {
				continue
			}

			// Attempt to join the cluster if this is a brand new node.
			if join {
				err := cli.Add(
					ctx,
					client.NodeInfo{ID: a.id, Address: a.address, Role: client.Spare})
				if err == nil {
					join = false
				} else {
					a.log(client.LogWarn, "join cluster: %v", err)
					delay = time.Second
					continue
				}

			}

			// Refresh our node store.
			servers, err := cli.Cluster(ctx)
			if err != nil {
				continue
			}
			a.store.Set(ctx, servers)

			// If we are starting up, let's see if we should
			// promote ourselves.
			if !ready {
				if err := a.maybeChangeRole(ctx, cli, servers); err != nil {
					a.log(client.LogWarn, "update our role: %v", err)
					continue
				}
			}

			// If we were just starting up, let's advertise
			// ourselves as ready.
			if !ready {
				ready = true
				close(a.readyCh)
			}

			delay = 30 * time.Second
		}
	}
}

// Possibly change our role at startup.
func (a *App) maybeChangeRole(ctx context.Context, cli *client.Client, nodes []client.NodeInfo) error {
	voters := 0
	role := client.NodeRole(-1)

	for _, node := range nodes {
		if node.ID == a.id {
			role = node.Role
		}
		if node.Role == client.Voter {
			voters++
		}
	}

	// If we are are in the list, it means we were removed, just do nothing.
	if role == -1 {
		return nil
	}

	// If we  already have the Voter role, or the cluster is still too small,
	// there's nothing to do.
	if role == client.Voter || len(nodes) < 3 {
		return nil
	}

	// Promote ourselves.
	a.debug("promote ourselves to voter")
	if err := cli.Assign(ctx, a.id, client.Voter); err != nil {
		return fmt.Errorf("assign voter role to ourselves: %v", err)
	}

	// Possibly try to promote another node as well if we've reached the 3 node
	// threshold.
	if voters == 1 {
		for _, node := range nodes {
			if node.ID == a.id || node.Role == client.Voter {
				continue
			}
			a.debug("promote %s to voter", node.Address)
			if err := cli.Assign(ctx, node.ID, client.Voter); err == nil {
				break
			}
		}
	}

	return nil
}

func (a *App) debug(format string, args ...interface{}) {
	a.log(client.LogDebug, format, args...)
}

func (a *App) info(format string, args ...interface{}) {
	a.log(client.LogInfo, format, args...)
}

func (a *App) warn(format string, args ...interface{}) {
	a.log(client.LogWarn, format, args...)
}

func (a *App) error(format string, args ...interface{}) {
	a.log(client.LogError, format, args...)
}

var driverIndex = 0
