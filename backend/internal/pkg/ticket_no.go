package pkg

import (
	"fmt"
	"sync/atomic"
	"time"
)

var ticketSeq int64

// GenerateTicketNo returns a ticket number in the format TK-YYYYMMDD-NNNN
func GenerateTicketNo() string {
	now := time.Now()
	date := now.Format("20060102")
	seq := atomic.AddInt64(&ticketSeq, 1)
	return fmt.Sprintf("TK-%s-%04d", date, seq)
}
