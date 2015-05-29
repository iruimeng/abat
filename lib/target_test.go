package abat

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNewTargetFrom(t *testing.T) {
	t.Parallel()

	lines := bytes.NewReader([]byte("GET http://www.baidu.com"))

	tgts, err := NewTargetFrom(lines, nil, nil)

	fmt.Println(tgts, err)
}
