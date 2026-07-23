package main

import "testing"

func TestFreshSyncDirFor(t *testing.T) {
	cases := []struct {
		server, want string
	}{
		{"https://fd.example.dev", "/home/u/discodrive-fd.example.dev"},
		{"https://fd.example.dev:8443", "/home/u/discodrive-fd.example.dev-8443"},
		{"http://192.168.1.10:9000", "/home/u/discodrive-192.168.1.10-9000"},
		{"not a url", "/home/u/discodrive-new"},
	}
	for _, c := range cases {
		if got := freshSyncDirFor("/home/u/discodrive", c.server); got != c.want {
			t.Errorf("freshSyncDirFor(%q) = %q, want %q", c.server, got, c.want)
		}
	}
}

func TestResolvePairDir(t *testing.T) {
	const def = "/home/u/discodrive"
	cases := []struct {
		name      string
		requested string
		explicit  bool
		oldServer string
		newServer string
		want      string
	}{
		{"first pairing keeps requested dir", def, false, "", "https://a.test", def},
		{"same server keeps requested dir", def, false, "https://a.test", "https://a.test", def},
		{"server change + explicit --dir is respected", "/data/x", true, "https://a.test", "https://b.test", "/data/x"},
		{"server change without --dir picks a fresh per-server dir", def, false, "https://a.test", "https://b.test", def + "-b.test"},
	}
	for _, c := range cases {
		if got := resolvePairDir(c.requested, c.explicit, c.oldServer, c.newServer, def); got != c.want {
			t.Errorf("%s: resolvePairDir = %q, want %q", c.name, got, c.want)
		}
	}
}
