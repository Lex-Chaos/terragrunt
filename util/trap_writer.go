package util

import (
	"bytes"
	"io"
)

type TrapWriter struct {
	Writer         io.Writer
	TargetMsgBytes []byte
	TrappedMsgs    []string
}

func (trap *TrapWriter) Write(msg []byte) (int, error) {
	if bytes.Contains(msg, trap.TargetMsgBytes) {
		trap.TrappedMsgs = append(trap.TrappedMsgs, string(msg))
		return len(msg), nil
	}
	return trap.Writer.Write(msg)
}
