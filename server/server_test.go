package server

import "testing"

func TestServe(t *testing.T) {
	Serve("127.0.0.1:1234")
}
