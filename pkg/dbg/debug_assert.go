//go:build debug

package dbg

func Assert(cond bool, msg ...any) {
	if !cond {
		if len(msg) > 0 {
			panic(msg[0])
		}
		panic("cond failed")
	}
}
