package protocol

// DO NOT EDIT
//
// This file was generated by ./schema.sh

import "fmt"


// DecodeFailure decodes a Failure response.
func DecodeFailure(response *Message) (code uint64, message string, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseFailure {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseFailure), mtype)
                return
	}

	code = response.getUint64()
	message = response.getString()

	return
}

// DecodeWelcome decodes a Welcome response.
func DecodeWelcome(response *Message) (heartbeatTimeout uint64, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseWelcome {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseWelcome), mtype)
                return
	}

	heartbeatTimeout = response.getUint64()

	return
}

// DecodeNodeLegacy decodes a NodeLegacy response.
func DecodeNodeLegacy(response *Message) (address string, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseNodeLegacy {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseNodeLegacy), mtype)
                return
	}

	address = response.getString()

	return
}

// DecodeNode decodes a Node response.
func DecodeNode(response *Message) (id uint64, address string, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseNode {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseNode), mtype)
                return
	}

	id = response.getUint64()
	address = response.getString()

	return
}

// DecodeNodes decodes a Nodes response.
func DecodeNodes(response *Message) (servers Nodes, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseNodes {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseNodes), mtype)
                return
	}

	servers = response.getNodes()

	return
}

// DecodeDb decodes a Db response.
func DecodeDb(response *Message) (id uint32, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseDb {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseDb), mtype)
                return
	}

	id = response.getUint32()
	response.getUint32()

	return
}

// DecodeStmt decodes a Stmt response.
func DecodeStmt(response *Message) (db uint32, id uint32, params uint64, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseStmt {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseStmt), mtype)
                return
	}

	db = response.getUint32()
	id = response.getUint32()
	params = response.getUint64()

	return
}

// DecodeEmpty decodes a Empty response.
func DecodeEmpty(response *Message) (err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseEmpty {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseEmpty), mtype)
                return
	}

	response.getUint64()

	return
}

// DecodeResult decodes a Result response.
func DecodeResult(response *Message) (result Result, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseResult {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseResult), mtype)
                return
	}

	result = response.getResult()

	return
}

// DecodeRows decodes a Rows response.
func DecodeRows(response *Message) (rows Rows, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseRows {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseRows), mtype)
                return
	}

	rows = response.getRows()

	return
}

// DecodeFiles decodes a Files response.
func DecodeFiles(response *Message) (files Files, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseFiles {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseFiles), mtype)
                return
	}

	files = response.getFiles()

	return
}

// DecodeMetadata decodes a Metadata response.
func DecodeMetadata(response *Message) (failureDomain uint64, weight uint64, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseMetadata {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseMetadata), mtype)
                return
	}

	failureDomain = response.getUint64()
	weight = response.getUint64()

	return
}

// DecodeMemory decodes a Memory response.
func DecodeMemory(response *Message) (mallocCount uint64, memoryUsed uint64, memoryWatermark uint64, logSize uint64, logN uint64, logRefs uint64, logLost uint64, logEnd uint64, logMissedSuffix uint64, logMissedPrefix uint64, logMissedRelease uint64, vfs uint64, err error) {
	mtype, _ := response.getHeader()

	if mtype == ResponseFailure {
		e := ErrRequest{}
		e.Code = response.getUint64()
		e.Description = response.getString()
                err = e
                return
	}

	if mtype != ResponseMemory {
		err = fmt.Errorf("decode %s: unexpected type %d", responseDesc(ResponseMemory), mtype)
                return
	}

	mallocCount = response.getUint64()
	memoryUsed = response.getUint64()
	memoryWatermark = response.getUint64()
	logSize = response.getUint64()
	logN = response.getUint64()
	logRefs = response.getUint64()
	logLost = response.getUint64()
	logEnd = response.getUint64()
	logMissedSuffix = response.getUint64()
	logMissedPrefix = response.getUint64()
	logMissedRelease = response.getUint64()
	vfs = response.getUint64()

	return
}
