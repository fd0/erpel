package erpel

import "testing"

const ruleViewTestGlobal = `
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
`

const ruleViewTestConfig = `
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
`

var ruleViewTests = []struct {
	template string
	result   []string
}{
	{
		template: "Jun  2 23:17:13 mail dovecot: lda(user@host.tld): sieve: msgid=<20160602211704.9125E5A063@localhost>: stored mail into mailbox 'INBOX'",
		result: []string{
			"{timestamp}",
			" mail dovecot: lda(",
			"[mailaddress]",
			"): sieve: msgid=<",
			"[msgid]",
			">: stored mail into mailbox 'INBOX'",
		},
	},
	{
		template: "Jun  2 23:17:13 mail dovecot: IMAP(username@domain.tld): Disconnected: Logged out bytes=123/123",
		result: []string{
			"{timestamp}",
			" mail dovecot: IMAP(",
			"[username]",
			"): Disconnected: Logged out bytes=",
			"[num]", "/", "[num]",
		},
	},
	{
		template: "foobar dovecot: IMAP(username@domain.tld): Disconnected: Logged out bytes=123/123",
		result: []string{
			"foobar dovecot: IMAP(",
			"[username]",
			"): Disconnected: Logged out bytes=",
			"[num]", "/", "[num]",
		},
	},
}

func parseRules(t testing.TB, data string) Rules {
	cfg, err := ParseConfig(ruleViewTestGlobal)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	rules, err := ParseRules(cfg.Fields, data)
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}

	if err = rules.Check(); err != nil {
		t.Fatalf("Check(): %v", err)
	}

	return rules
}

func TestRuleView(t *testing.T) {
	rules := parseRules(t, ruleViewTestConfig)

	for i, test := range ruleViewTests {
		res := View(rules, test.template)

		max := len(test.result)
		if len(res) != max {
			t.Errorf("test %d: unexpected number of fields returned, want %d, got %d, want:\n  %q\ngot:\n  %q",
				i, max, len(res), test.result, res)
		}

		if len(res) < max {
			max = len(res)
		}

		for j := 0; j < max; j++ {
			s := res[j].String()
			if s != test.result[j] {
				t.Errorf("test %d: unexpected output for field %d: want %q, got %q",
					i, j, test.result[j], s)
			}
		}
	}
}
