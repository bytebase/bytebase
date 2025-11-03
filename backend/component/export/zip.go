package export

import (
	"io"
	"time"

	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
)

// CreateZipFileHeader creates a zip file header with optional password protection.
func CreateZipFileHeader(filename string, password string) *zip.FileHeader {
	fh := &zip.FileHeader{
		Name:   filename,
		Method: zip.Deflate,
	}
	fh.ModifiedDate, fh.ModifiedTime = timeToMsDosTime(time.Now())
	if password != "" {
		fh.SetPassword(password)
	}
	return fh
}

// WriteZipEntry writes content to a zip file entry.
func WriteZipEntry(zipw *zip.Writer, filename string, content []byte, password string) error {
	fh := CreateZipFileHeader(filename, password)
	writer, err := zipw.CreateHeader(fh)
	if err != nil {
		return errors.Wrapf(err, "failed to create zip entry for %s", filename)
	}
	if _, err := writer.Write(content); err != nil {
		return errors.Wrapf(err, "failed to write zip entry for %s", filename)
	}
	return nil
}

// CreateZipWriter creates a zip writer for the given writer with a file header.
func CreateZipWriter(zipw *zip.Writer, filename string, password string) (io.Writer, error) {
	fh := CreateZipFileHeader(filename, password)
	writer, err := zipw.CreateHeader(fh)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create zip entry for %s", filename)
	}
	return writer, nil
}

// timeToMsDosTime converts a time.Time to an MS-DOS date and time.
// This is a modified copy for github.com/alexmullins/zip/struct.go because the package has a bug,
// it will convert the time to UTC time and drop the timezone.
func timeToMsDosTime(t time.Time) (uint16, uint16) {
	fDate := uint16(t.Day() + int(t.Month())<<5 + (t.Year()-1980)<<9)
	fTime := uint16(t.Second()/2 + t.Minute()<<5 + t.Hour()<<11)
	return fDate, fTime
}
