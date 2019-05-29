/* {{{ Copyright (c) Paul R. Tagliamonte <paultag@debian.org>, 2015
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE. }}} */

package deb // import "github.com/ebikt/go-debian/deb"

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

// ArEntry {{{

// Container type to access the different parts of a Debian `ar(1)` Archive.
//
// The most interesting parts of this are the `Name` attribute, Data
// `io.Reader`, and the Tarfile helpers. This will allow the developer to
// programmatically inspect the information inside without forcing her to
// unpack the .deb to the filesystem.
type ArEntry struct {
	Name      string
	Timestamp int64
	OwnerID   int64
	GroupID   int64
	FileMode  string
	Size      int64
	Data      io.Reader
}

// }}}

// Ar {{{

// This struct encapsulates a Debian .deb flavored `ar(1)` archive.
type Ar struct {
	in         io.Reader
	lastReader *io.Reader
	offset     bool
}

// LoadAr {{{

// Load an Ar archive reader from an io.Reader
func LoadAr(in io.Reader) (*Ar, error) {
	if err := checkAr(in); err != nil {
		return nil, err
	}
	debFile := Ar{in: in}
	return &debFile, nil
}

// }}}

// Next {{{

// Function to jump to the next file in the Debian `ar(1)` archive, and
// return the next member.
func (d *Ar) Next() (*ArEntry, error) {
	if d.lastReader != nil {
		/* Before we do much more, let's empty out the reader, since we
		 * can't be sure of our position in the reader until the LimitReader
		 * is empty */
		if _, err := io.Copy(ioutil.Discard, *d.lastReader); err != nil {
			return nil, err
		}
		if d.offset {
			/* .ar archives align on 2 byte boundaries, so if we're odd, go
			 * ahead and read another byte. If we get an io.EOF, it's fine
			 * to return it. */
			_, err := d.in.Read(make([]byte, 1))
			if err != nil {
				return nil, err
			}
		}
	}

	line := make([]byte, 60)
	pos := 0

	var err error
	for pos < 60 {
		var count int
		count, err = d.in.Read(line[pos:])
		if (err == nil) || (err == io.EOF){
			pos = pos + count
		}
		if (err == io.EOF) { break }
		if (err != nil) { return nil, err }
	}
	if err == io.EOF {
		if pos == 0 || (pos == 1 && line[0] == '\n') {
			return nil, err
		}
	}
	if pos < 60 {
		return nil, fmt.Errorf("Caught a short read at the end")
	}
	entry, err := parseArEntry(line)
	if err != nil {
		return nil, err
	}

	entry.Data = io.LimitReader(d.in, entry.Size)
	d.lastReader = &entry.Data
	d.offset = (entry.Size % 2) == 1

	return entry, nil
}

// }}}

// toDecimal {{{

// Take a byte array, and return an int64
func toDecimal(input []byte) (int64, error) {
	stream := strings.TrimSpace(string(input))
	out, err := strconv.Atoi(stream)
	return int64(out), err
}

// }}}

// }}}

// AR Format Hackery {{{

// parseArEntry {{{

// Take the AR format line, and create an ArEntry (without .Data set)
// to be returned to the user later.
//
// +-------------------------------------------------------
// | Offset  Length  Name                         Format
// +-------------------------------------------------------
// | 0       16      File name                    ASCII
// | 16      12      File modification timestamp  Decimal
// | 28      6       Owner ID                     Decimal
// | 34      6       Group ID                     Decimal
// | 40      8       File mode                    Octal
// | 48      10      File size in bytes           Decimal
// | 58      2       File magic                   0x60 0x0A
//
func parseArEntry(line []byte) (*ArEntry, error) {
	if len(line) != 60 {
		return nil, fmt.Errorf("Malformed file entry line length")
	}

	if line[58] != 0x60 && line[59] != 0x0A {
		return nil, fmt.Errorf("Malformed file entry line endings")
	}

	entry := ArEntry{
		// Found a valid deb packages with trailing slash in ar names.
		// According to wikipedia, that is System V extension. -- Ebik.
		Name:     strings.TrimSuffix(
				strings.TrimSpace(string(line[0:16])), "/",
			  ),
		FileMode: strings.TrimSpace(string(line[48:58])),
	}

	for target, value := range map[*int64][]byte{
		&entry.Timestamp: line[16:28],
		&entry.OwnerID:   line[28:34],
		&entry.GroupID:   line[34:40],
		&entry.Size:      line[48:58],
	} {
		intValue, err := toDecimal(value)
		if err != nil {
			return nil, err
		}
		*target = intValue
	}

	return &entry, nil
}

// }}}

// checkAr {{{

// Given a brand spank'n new os.File entry, go ahead and make sure it looks
// like an `ar(1)` archive, and not some random file.
func checkAr(reader io.Reader) error {
	header := make([]byte, 8)
	pos := 0
	for pos < 8 {
		count, err := reader.Read(header[pos:])
		if  err == io.EOF {
			if count+pos == 0  {
				return fmt.Errorf("File is empty.")
			} else {
				return fmt.Errorf("Header too short for 'ar' file.")
			}
		}
		if  err != nil {
			return err
		}
		pos = pos + count
	}
	if string(header) != "!<arch>\n" {
		return fmt.Errorf("Header doesn't look as 'ar' file.")
	}
	return nil
}

// }}}

// }}}

// vim: foldmethod=marker
