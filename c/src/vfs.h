#ifndef DQLITE_VFS_H
#define DQLITE_VFS_H

#include <pthread.h>

#include <sqlite3.h>

/* Hold content for a single page or frame in a volatile file. */
struct dqlite__vfs_page {
	void *buf;        /* Content of the page. */
	void *hdr;        /* Page header (only for WAL pages). */
	void *dirty_mask; /* Bit mask of dirty buf bytes to be re-written (only for
	                     WAL pages) */
	int   dirty_mask_size; /* Number of bytes in the dirty_mask array. */
	void *dirty_buf; /* List of dirty buf bytes, one for each bit with value 1 in
	                    dirty_mask. */
};

/* Hold content for a single file in the volatile file system. */
struct dqlite__vfs_content {
	char *                    filename;  /* Name of the file. */
	void *                    hdr;       /* File header (only for WAL files). */
	struct dqlite__vfs_page **pages;     /* Pointers to all pages in the file. */
	int                       pages_len; /* Number of pages in the file. */
	int                       page_size; /* Size of page->buf for each page. */

	int    refcount;        /* Number of open FDs referencing this file. */
	int    type;            /* Content type (either main db or WAL). */
	void **shm_regions;     /* Pointers to shared memory regions. */
	int    shm_regions_len; /* Number of shared memory regions. */
	int    shm_refcount;    /* Number of opened files using the shared memory. */

	struct dqlite__vfs_content *wal; /* WAL file content (for database files). */

	/* For database files, number of ongoing transations across all db
	 * connections using this database. Used to decide whether it's safe to
	 * issue a checkpoint after a commit. */
	int tx_refcount;
};

// Root of the volatile file system. Contains pointers to the content
// of all files that were created.
struct dqlite__vfs_root {
	struct dqlite__vfs_content **contents;     /* Files content */
	int                          contents_len; /* Number of files */
	pthread_mutex_t              mutex;        /* Serialize to access */
	int                          error;        /* Last error occurred. */
};

struct dqlite__vfs_file {
	sqlite3_file base; /* Base class. Must be first. */
	struct dqlite__vfs_root
	    *root; /* Pointer to our volatile VFS instance data. */
	struct dqlite__vfs_content *content; /* Handle to the file content. */
};

#endif /* DQLITE_VFS_H */
