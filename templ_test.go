package main

import "testing"

func TestTitle(t *testing.T) {
	for i, tt := range []struct {
		input, want string
	}{
		{"foo cpu baz", "FooCPUBaz"},
		{"test Id", "TestID"},
		{"json and html", "JSONAndHTML"},
	} {
		if got := lintTitle(tt.input); got != tt.want {
			t.Errorf("[%d] title(%q) = %q, want %q", i, tt.input, got, tt.want)
		}
	}
}

func TestSquish(t *testing.T) {
	for i, tt := range []struct {
		input, want string
	}{
		{"Foo CPU Baz", "FooCPUBaz"},
		{"Test ID", "TestID"},
		{"JSON And HTML", "JSONAndHTML"},
	} {
		if got := squish(tt.input); got != tt.want {
			t.Errorf("[%d] squish(%q) = %q, want %q", i, tt.input, got, tt.want)
		}
	}
}

func TestReplace(t *testing.T) {
	for i, tt := range []struct {
		input, want string
	}{
		{"foo Cpu baz", "foo CPU baz"},
		{"test Id", "test ID"},
		{"Json and Html", "JSON and HTML"},
	} {
		if got := initialisms.Replace(tt.input); got != tt.want {
			t.Errorf("[%d] replace(%q) = %q, want %q", i, tt.input, got, tt.want)
		}
	}

	c := len(initialismPairs)

	for i := 0; i < c; i++ {
		input, want := initialismPairs[i], initialismPairs[i+1]

		if got := initialisms.Replace(input); got != want {
			t.Errorf("[%d] replace(%q) = %q, want %q", i, input, got, want)
		}

		i++
	}
}
