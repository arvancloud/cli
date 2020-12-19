package utl

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultErrorExitCode = 1
)

// #TODO improve check error to print better

// fatal prints the message (if provided) and then exits.
func fatalErrHandler(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

// CheckErr prints a user friendly error to STDERR and exits with a non-zero
// exit code. Unrecognized errors will be printed with an "error: " prefix.
func CheckErr(err error) {
	checkErr(err, fatalErrHandler)
}

// checkErr formats a given error as a string and calls the passed handleErr
func checkErr(err error, handleErr func(string, int)) {
	if err == nil {
		return
	}
	fmt.Println(err)
	handleErr("", DefaultErrorExitCode)
}

// ReadInput prints explain and repeat printing inputExplain to out and reads a string from in.
//   If input is empty and defaultVal is set returns default value
//   If defaultVal is not set, tries to validate input using validate
func ReadInput(inputExplain, defaultVal string, out io.Writer, in io.Reader, validate func(string) (bool, error)) string {
	reader := bufio.NewReader(in)
	for {
		_, err := fmt.Fprint(out, inputExplain)
		if err != nil {
			log.Println(err)
		}
		i, err := reader.ReadString('\n')
		if err != nil {
			_, err := fmt.Fprintf(out, "Error: %s\n", err.Error())
			if err != nil {
				log.Println(err)
			}
		} else {
			i = strings.TrimSpace(i)
			if len(i) == 0 && len(defaultVal) > 0 {
				return defaultVal
			}
			valid, err := validate(i)
			if valid {
				return i
			}
			_, err = fmt.Fprintf(out, "Error: %s\n", err.Error())
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func Untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}
		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		}
	}
}

func Unzip(dest string, rc io.ReadCloser) ([]string, error) {
	tmpFile := filepath.Join(os.TempDir(), "arvan_cli.zip")
	bytes, err := ioutil.ReadAll(rc)
	err = ioutil.WriteFile(tmpFile, bytes, 0644)
	if err != nil {
		panic(err)
	}

	var filenames []string

	r, err := zip.OpenReader(tmpFile)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
