package file

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"time"
)

// OSFile is a Source backed by an *os.Root-sandboxed file.
// Path traversal is structurally impossible: all access goes through
// os.Root.Open which restricts paths to within its root directory.
type OSFile struct {
	root    *os.Root
	rel     string
	size    int64
	modTime time.Time
	ct      string
}

// OpenOSFile opens rel within root as a Source.
// Returns an error if the file does not exist or is a directory.
func OpenOSFile(root *os.Root, rel string) (*OSFile, error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("file: open %q: %w", rel, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("file: stat %q: %w", rel, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("file: %q is a directory", rel)
	}

	ct := mime.TypeByExtension(filepath.Ext(rel))
	if ct == "" {
		ct = "application/octet-stream"
	}

	return &OSFile{
		root:    root,
		rel:     rel,
		size:    fi.Size(),
		modTime: fi.ModTime(),
		ct:      ct,
	}, nil
}

func (o *OSFile) Size() int64         { return o.size }
func (o *OSFile) ModTime() time.Time  { return o.modTime }
func (o *OSFile) ContentType() string { return o.ct }
func (o *OSFile) Name() string        { return filepath.Base(o.rel) }

func (o *OSFile) ReadAt(p []byte, off int64) (int, error) {
	f, err := o.root.Open(o.rel)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// *os.File implements io.ReaderAt directly.
	return f.ReadAt(p, off)
}
