package erpel

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func tempfile(t *testing.T) string {
	f, err := ioutil.TempFile("", "erpel-test-")
	if err != nil {
		t.Fatalf("TempFile(): %v", err)
	}

	name := f.Name()

	if err = f.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	return name
}

func log(t *testing.T, filename string, data string) Marker {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("open(): %v", err)
	}

	if _, err = f.Write([]byte(data)); err != nil {
		t.Errorf("write(): %v", err)
	}

	m, err := Position(f)
	if err != nil {
		t.Errorf("Position(): %v", err)
	}

	if err = f.Close(); err != nil {
		t.Fatalf("close(): %v", err)
	}

	return m
}

func rm(t *testing.T, filename string) {
	if err := os.Remove(filename); err != nil {
		t.Errorf("removing %q failed: %v", filename, err)
	}
}

func writeMessages(t *testing.T, filename string, m []string) []Marker {
	var markers []Marker
	for _, msg := range messages {
		m := log(t, filename, msg)
		markers = append(markers, m)
	}

	return markers
}

func readRemainingData(t *testing.T, filename string, m Marker) []byte {
	fd, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open(%v): %v", filename, err)
	}

	if err = m.Seek(fd); err != nil {
		t.Fatalf("Marker.Seek(): %v", err)
	}

	buf, err := ioutil.ReadAll(fd)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if err = fd.Close(); err != nil {
		t.Fatalf("close(%v): %v", filename, err)
	}

	return buf
}

var messages = []string{
	"foobar baz message\n",
	"Jun 30 21:57:10 mopped sudo[19517]: pam_unix(sudo:session): session opened for user root by fd0(uid=0)\n",
	"foobar message2\n",
	"Jun 30 21:57:15 mopped sudo[19517]: pam_unix(sudo:session): session closed for user root\n",
}

func TestMarker(t *testing.T) {
	f := tempfile(t)
	t.Logf("using tempfile %v", f)

	markers := writeMessages(t, f, messages)
	for i, m := range markers {
		var data []byte

		for j := i + 1; j < len(messages); j++ {
			data = append(data, []byte(messages[j])...)
		}

		buf := readRemainingData(t, f, m)

		if !bytes.Equal(buf, data) {
			t.Errorf("marker %d returned wrong data, want:\n  %q\ngot:\n  %q", i, data, buf)
		}
	}

	rm(t, f)
}

func writeFile(t *testing.T, filename string, data []byte) {
	fd, err := os.Create(filename)
	if err != nil {
		t.Fatalf("create() %v", err)
	}

	_, err = fd.Write(data)
	if err != nil {
		t.Fatalf("write() %v", err)
	}

	if err = fd.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}
}

func mv(t *testing.T, from, to string) {
	if err := os.Rename(from, to); err != nil {
		t.Errorf("move %v -> %v failed: %v", from, to, err)
	}
}

func TestMarkerNewFile(t *testing.T) {
	f := tempfile(t)
	t.Logf("using tempfile %v", f)

	markers := writeMessages(t, f, messages)
	data := []byte(strings.Join(messages, ""))

	mv(t, f, f+".1")
	writeFile(t, f, data)

	fi, err := os.Stat(f)
	if err != nil {
		t.Fatalf("stat(): %v", err)
	}
	t.Logf("stat(%v): %#v", f, fi)

	for i, m := range markers {
		buf := readRemainingData(t, f, m)

		t.Logf("marker %d: %v", i, m)

		if !bytes.Equal(buf, data) {
			t.Errorf("marker %d returned wrong data, want:\n  %q\ngot:\n  %q", i, data, buf)
		}
	}

}
