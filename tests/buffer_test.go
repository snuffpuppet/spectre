package buffer_test

import (
	"github.com/snuffpuppet/spectre/pcm"
	"reflect"
	"testing"
)

const SAMPLE_RATE = 4096

func TestIntFormat(t *testing.T) {
	b := pcm.NewIntBuffer(SAMPLE_RATE)

	// Test that it is actually an Int buffer
	if b.DataFormat() != pcm.FMT_INT16 {
		t.Errorf("IntBuffer not identifying as %s\n", pcm.FMT_INT16)
	}

	if _, ok := b.Frame().([]int16); !ok {
		t.Errorf("IntBuffer raw data tyoe is not %s, is actually %s\n", pcm.FMT_INT16, reflect.TypeOf(b.Frame()))
	}
}

func TestIntData(t *testing.T) {
	//b := pcm.NewIntBuffer(SAMPLE_RATE)

}
