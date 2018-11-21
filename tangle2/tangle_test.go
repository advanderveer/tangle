package tangle_test

import (
	"bytes"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	tangle "tangle/tangle2"

	test "github.com/advanderveer/go-test"
)

func TestTipSelection(t *testing.T) {
	tngl := tangle.NewTangle()
	g := tngl.Genesis()
	test.Equals(t, 2, len(g))
	test.Equals(t, uint64(1), g[0])
	test.Equals(t, uint64(2), g[1])

	tips := tngl.SelectTips(2, 100)
	test.Equals(t, uint64(1), tips[0])
	test.Equals(t, uint64(2), tips[1])
}

func drawPNG(t *testing.T, buf io.Reader, name string) {
	f, err := os.Create(name)
	test.Ok(t, err)
	defer f.Close()

	cmd := exec.Command("dot", "-Tpng")
	cmd.Stdin = buf
	cmd.Stdout = f
	test.Ok(t, cmd.Run())
}

func nextTime(rnd *rand.Rand, r float64) float64 {
	return -math.Log(1.0-rnd.Float64()) / r
}

func timeline(seed int64, n int, λ float64, every time.Duration) (c chan time.Time) {
	rnd := rand.New(rand.NewSource(seed))
	c = make(chan time.Time)
	go func() {
		defer close(c)
		for i := 0; i < n; i++ {
			d := time.Duration(float64(every) * nextTime(rnd, λ))
			c <- <-time.After(d)
		}
	}()

	return
}

func TestGraphDrawing(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	tngl := tangle.NewTangle()
	rnd := rand.New(rand.NewSource(42))

	n := 120
	λ := 2.1
	u := time.Millisecond * 10

	var wg sync.WaitGroup
	for range timeline(42, n, λ, u) {
		wg.Add(1)
		d := time.Duration(rnd.Int63n(int64(u)))

		go func() {
			defer wg.Done()

			tips := tngl.SelectTips(2, 100)      //find suitable tips
			time.Sleep(d)                        //network latency
			tngl.ReceiveBlock([]byte{}, tips...) //submit block

		}()
	}

	wg.Wait()
	err := tngl.Draw(buf)
	test.Ok(t, err)

	drawPNG(t, buf, "basic_test.png")
}
