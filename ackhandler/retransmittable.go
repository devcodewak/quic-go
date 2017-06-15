package ackhandler

import (
	"github.com/lucas-clemente/quic-go/frames"
)

// Returns a new slice with all non-retransmittable frames deleted.
func stripNonRetransmittableFrames(fs []frames.Frame) []frames.Frame {
	res := make([]frames.Frame, 0, len(fs))
	for _, f := range fs {
		if isFrameRetransmittable(f) {
			res = append(res, f)
		}
	}
	return res
}

func isFrameRetransmittable(f frames.Frame) bool {
	switch f.(type) {
	case *frames.StopWaitingFrame:
		return false
	case *frames.AckFrame:
		return false
	default:
		return true
	}
}
