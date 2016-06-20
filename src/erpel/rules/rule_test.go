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

------------------

Jun  2 23:17:13 mail dovecot: lda(user@host.tld): sieve: msgid=<20160602211704.9125E5A063@localhost>: stored mail into mailbox 'INBOX'
Jun  2 23:17:13 avalon dovecot: IMAP(username@domain): Disconnected: Logged out bytes=123/123
Jun  2 23:17:13 mail dovecot: imap-login: Disconnected (no auth attempts in 0 secs): user=<>, rip=1.2.3.4, lip=1.2.3.4, TLS handshaking: Disconnected, session=<O3h6IVI0sQBQu1D7>

--------------

Jun  2 23:17:13 mail dovecot: lda(me@domain.de): sieve: msgid=<20160602211704.9125E5A063@graphite.x.net>: stored mail into mailbox 'INBOX'
Jun  2 23:17:14 avalon dovecot: IMAP(foobar): Disconnected: Logged out bytes=1152/16042
Jun  2 23:17:17 mail dovecot: imap-login: Disconnected (no auth attempts in 0 secs): user=<>, rip=123.23.123.1, lip=192.168.0.1, TLS handshaking: Disconnected, session=<O3h6IVI0sQBQu1D7>
Jun  2 23:17:17 mail dovecot: imap-login: Login: user=<me@domain.de>, method=PLAIN, rip=1234:1234::1234, lip=2a01:4f8::1234:1, mpid=32650, TLS, session=<0Xl9IVI0GAAqAQWYoAGvEHy/uaNfPKFR>
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
			},
			Templates: []string{
				`Jun  2 23:17:13 mail dovecot: lda(user@host.tld): sieve: msgid=<20160602211704.9125E5A063@localhost>: stored mail into mailbox 'INBOX'`,
				`Jun  2 23:17:13 avalon dovecot: IMAP(username@domain): Disconnected: Logged out bytes=123/123`,
				`Jun  2 23:17:13 mail dovecot: imap-login: Disconnected (no auth attempts in 0 secs): user=<>, rip=1.2.3.4, lip=1.2.3.4, TLS handshaking: Disconnected, session=<O3h6IVI0sQBQu1D7>`,
			},
			Samples: []string{
				`Jun  2 23:17:13 mail dovecot: lda(me@domain.de): sieve: msgid=<20160602211704.9125E5A063@graphite.x.net>: stored mail into mailbox 'INBOX'`,
				`Jun  2 23:17:14 avalon dovecot: IMAP(foobar): Disconnected: Logged out bytes=1152/16042`,
				`Jun  2 23:17:17 mail dovecot: imap-login: Disconnected (no auth attempts in 0 secs): user=<>, rip=123.23.123.1, lip=192.168.0.1, TLS handshaking: Disconnected, session=<O3h6IVI0sQBQu1D7>`,
				`Jun  2 23:17:17 mail dovecot: imap-login: Login: user=<me@domain.de>, method=PLAIN, rip=1234:1234::1234, lip=2a01:4f8::1234:1, mpid=32650, TLS, session=<0Xl9IVI0GAAqAQWYoAGvEHy/uaNfPKFR>`,
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

		if !reflect.DeepEqual(rules, test.rules) {
			t.Errorf("test %v: rules are not equal:\n  want:\n    %#v\n  got:\n    %#v", i, test.rules, rules)
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
			t.Fatal(err)
		}

		_, err = ParseRules(string(buf))
		if err != nil {
			t.Fatalf("parsing rules file %v failed: %v", file, err)
		}
	}
}
