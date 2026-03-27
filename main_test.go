package main

import "testing"

func TestIsLikelyGenreEditorialImage(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{title: "File:Ambient_music.jpg", want: true},
		{title: "File:Padlock.svg", want: false},
		{title: "File:Wikipedia-logo-v2.svg", want: false},
	}
	for _, tc := range cases {
		if got := isLikelyGenreEditorialImage(tc.title); got != tc.want {
			t.Fatalf("isLikelyGenreEditorialImage(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
}
