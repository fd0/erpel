package erpel

import (
	"bytes"
	"io/ioutil"
	"os"
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

var messages = []string{
	"foobar baz message\n",
	"Jun 30 21:57:10 mopped sudo[19517]: pam_unix(sudo:session): session opened for user root by fd0(uid=0)\n",
	"foobar message2\n",
	"Jun 30 21:57:15 mopped sudo[19517]: pam_unix(sudo:session): session closed for user root\n",
}

func TestMarker(t *testing.T) {
	f := tempfile(t)
	t.Logf("using tempfile %v", f)

	var markers []Marker
	for _, msg := range messages {
		m := log(t, f, msg)
		markers = append(markers, m)
	}

	for i, m := range markers {
		var data []byte

		for j := i + 1; j < len(messages); j++ {
			data = append(data, []byte(messages[j])...)
		}

		fd, err := os.Open(f)
		if err != nil {
			t.Fatalf("open(%v): %v", f, err)
		}

		if err = m.Seek(fd); err != nil {
			t.Fatalf("Marker.Seek(): %v", err)
		}

		buf, err := ioutil.ReadAll(fd)
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		if err = fd.Close(); err != nil {
			t.Fatalf("close(%v): %v", f, err)
		}

		if !bytes.Equal(buf, data) {
			t.Errorf("marker %d returned wrong data, want:\n  %q\ngot:\n  %q", i, data, buf)
		}
	}

	rm(t, f)
}
