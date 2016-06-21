package rules

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

var testRulesFiles = []struct {
	data  string
	rules Rules
}{
	{
		data: `
# A field consists of a name and a template (to insert the field).
field timestamp {
    template = 'Jun  2 23:17:13'
    pattern = '\w{3}  ?\d{1,2} \d{2}:\d{2}:\d{2}'
}

# A field can also list examples, these must match the defined pattern.
field IP {
    template = '1.2.3.4'
    # this matches IPv4 and IPv6 addresses
    pattern = '(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4})'

    samples = ['192.168.100.1', '2003::feff:1234']
}

field msgid {
	template = '20160602211704.9125E5A063@localhost'
	pattern = '[a-zA-Z0-9.=@/-]+'
}

field mailaddress {
    template = 'user@host.tld'
    pattern = '[a-zA-Z0-9_+.-]+@[a-zA-Z0-9_+.-]+\.[a-zA-Z0-9_+.-]+'
}

field username {
    template = 'username@domain.tld'
    pattern = '[a-zA-Z0-9_+.-]+(@[a-zA-Z0-9_+.-]+\.[a-zA-Z0-9_+.-]+)?'
}

field num {
    template = '123'
    pattern = '\d+'
}

------------------

Jun  2 23:17:13 mail dovecot: lda(user@host.tld): sieve: msgid=<20160602211704.9125E5A063@localhost>: stored mail into mailbox 'INBOX'
Jun  2 23:17:13 mail dovecot: IMAP(username@domain.tld): Disconnected: Logged out bytes=123/123

--------------

Jun  2 23:17:18 mail dovecot: lda(me@domain.de): sieve: msgid=<20160602211704.9125E5A063@graphite.x.net>: stored mail into mailbox 'INBOX'
Jun  2 23:17:22 mail dovecot: IMAP(foobar): Disconnected: Logged out bytes=1152/16042
`,
		rules: Rules{
			Fields: map[string]Field{
				"timestamp": Field{
					Template: "Jun  2 23:17:13",
					Pattern:  regexp.MustCompile(`\w{3}  ?\d{1,2} \d{2}:\d{2}:\d{2}`),
				},
				"IP": Field{
					Template: "1.2.3.4",
					Pattern:  regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4})`),
					Samples:  []string{"192.168.100.1", "2003::feff:1234"},
				},
				"msgid": Field{
					Template: "20160602211704.9125E5A063@localhost",
					Pattern:  regexp.MustCompile(`[a-zA-Z0-9.=@/-]+`),
				},
				"mailaddress": Field{
					Template: "user@host.tld",
					Pattern:  regexp.MustCompile(`[a-zA-Z0-9_+.-]+@[a-zA-Z0-9_+.-]+\.[a-zA-Z0-9_+.-]+`),
				},
				"username": Field{
					Template: "username@domain.tld",
					Pattern:  regexp.MustCompile(`[a-zA-Z0-9_+.-]+(@[a-zA-Z0-9_+.-]+\.[a-zA-Z0-9_+.-]+)?`),
				},
				"num": Field{
					Template: "123",
					Pattern:  regexp.MustCompile(`\d+`),
				},
			},
			Templates: []string{
				`Jun  2 23:17:13 mail dovecot: lda(user@host.tld): sieve: msgid=<20160602211704.9125E5A063@localhost>: stored mail into mailbox 'INBOX'`,
				`Jun  2 23:17:13 mail dovecot: IMAP(username@domain.tld): Disconnected: Logged out bytes=123/123`,
			},
			Samples: []string{
				`Jun  2 23:17:18 mail dovecot: lda(me@domain.de): sieve: msgid=<20160602211704.9125E5A063@graphite.x.net>: stored mail into mailbox 'INBOX'`,
				`Jun  2 23:17:22 mail dovecot: IMAP(foobar): Disconnected: Logged out bytes=1152/16042`,
			},
		},
	},
}

func TestParse(t *testing.T) {
	for i, test := range testRulesFiles {
		rules, err := ParseRules(test.data)
		if err != nil {
			t.Errorf("test %v: parse failed: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(rules.Templates, test.rules.Templates) {
			t.Errorf("test %v: templates are not equal:\n  want:\n    %#v\n  got:\n    %#v",
				i, test.rules.Templates, rules.Templates)
		}

		if !reflect.DeepEqual(rules.Samples, test.rules.Samples) {
			t.Errorf("test %v: samples are not equal:\n  want:\n    %#v\n  got:\n    %#v",
				i, test.rules.Samples, rules.Samples)
		}

		names := make(map[string]struct{})
		for name := range rules.Fields {
			names[name] = struct{}{}
		}
		for name := range test.rules.Fields {
			names[name] = struct{}{}
		}

		for name := range names {
			if !rules.Fields[name].Equals(test.rules.Fields[name]) {
				t.Errorf("   field %v is not equal:\n  want:\n     %+v\n  got:\n     %+v",
					name, test.rules.Fields[name], rules.Fields[name])
			}
		}
	}
}

var testUnquoteString = []struct {
	data   string
	result string
}{
	{
		data:   "",
		result: "",
	},
	{
		data:   `"foobar"`,
		result: "foobar",
	},
	{
		data:   `"foo\nbar"`,
		result: "foo\nbar",
	},
	{
		data:   `"foo\x0abar"`,
		result: "foo\nbar",
	},
	{
		data:   `"foo\u000abar"`,
		result: "foo\nbar",
	},
	{
		data:   `"foo\"bar"`,
		result: `foo"bar`,
	},
	{
		data:   `'foo bar '`,
		result: "foo bar ",
	},
	{
		data:   `'foo \'bar '`,
		result: "foo 'bar ",
	},
	{
		data:   "`foo'\"bar `",
		result: "foo'\"bar ",
	},
}

func TestUnquoteString(t *testing.T) {
	for i, test := range testUnquoteString {
		s, err := unquoteString(test.data)
		if err != nil {
			t.Errorf("test %d: unquoteString(%q) return error: %v", i, test.data, err)
			continue
		}

		if s != test.result {
			t.Errorf("test %d: unquoteString(%q) return wrong result: want %q, got %q", i, test.data, test.result, s)
			continue
		}
	}
}

var testUnquoteList = []struct {
	data   string
	result []string
}{
	{
		`[]`,
		[]string{},
	},
	{
		`["foo", "bar", 'baz']`,
		[]string{"foo", "bar", "baz"},
	},
	{
		`["f"]`,
		[]string{"f"},
	},
	{
		"['f', `x`]",
		[]string{"f", "x"},
	},
}

func TestUnquoteList(t *testing.T) {
	for i, test := range testUnquoteList {
		res, err := unquoteList(test.data)
		if err != nil {
			t.Errorf("test %d failed: %v (data %q)", i, err, test.data)
			continue
		}

		if !reflect.DeepEqual(test.result, res) {
			t.Errorf("test %d failed: want %#v, got %#v", i, test.result, res)
		}
	}
}

func TestSampleConfig(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*"))
	if err != nil {
		t.Fatalf("unable to list directory testdata/: %v", err)
	}

	for _, file := range files {
		buf, err := ioutil.ReadFile(file)
		if err != nil {
			t.Error(err)
			continue
		}

		rules, err := ParseRules(string(buf))
		if err != nil {
			t.Errorf("parsing rules file %v failed: %v", file, err)
			continue
		}

		if err := rules.Check(); err != nil {
			t.Errorf("checking rules in file %v failed: %v", file, err)
			continue
		}
	}
}
