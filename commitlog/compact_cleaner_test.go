package commitlog_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/briansorahan/jocko/commitlog"
	"github.com/briansorahan/jocko/protocol"
)

func TestCompactCleaner(t *testing.T) {
	req := require.New(t)
	var err error

	var msgSets []commitlog.MessageSet
	msgSets = append(msgSets, newMessageSet(0, &protocol.Message{
		Key:       []byte("travisjeffery"),
		Value:     []byte("one tj"),
		MagicByte: 2,
		Timestamp: time.Now(),
	}))

	msgSets = append(msgSets, newMessageSet(1, &protocol.Message{
		Key:       []byte("another"),
		Value:     []byte("one another"),
		MagicByte: 2,
		Timestamp: time.Now(),
	}))

	msgSets = append(msgSets, newMessageSet(2, &protocol.Message{
		Key:       []byte("travisjeffery"),
		Value:     []byte("two tj"),
		MagicByte: 2,
		Timestamp: time.Now(),
	}))

	msgSets = append(msgSets, newMessageSet(3, &protocol.Message{
		Key:       []byte("again another"),
		Value:     []byte("again another"),
		MagicByte: 2,
		Timestamp: time.Now(),
	}))

	path, err := ioutil.TempDir("", "commitlog-delete-cleaner")
	req.NoError(err)
	defer os.RemoveAll(path)

	l := setupWithOptions(t, commitlog.Options{
		MaxSegmentBytes: int64(len(msgSets[0]) + len(msgSets[1])),
		MaxLogBytes:     1000,
	})
	defer cleanup(t, l)

	for _, msgSet := range msgSets {
		_, err = l.Append(msgSet)
		require.NoError(t, err)
	}

	segments := l.Segments()
	req.Equal(2, len(l.Segments()))
	segment := segments[0]

	scanner := commitlog.NewSegmentScanner(segment)
	ms, err := scanner.Scan()
	require.NoError(t, err)
	require.Equal(t, msgSets[0], ms)

	cc := commitlog.NewCompactCleaner()
	cleaned, err := cc.Clean(segments)
	req.NoError(err)
	req.Equal(2, len(cleaned))

	scanner = commitlog.NewSegmentScanner(cleaned[0])

	var count int
	for {
		ms, err = scanner.Scan()
		if err != nil {
			break
		}
		req.Equal(1, len(ms.Messages()))
		req.Equal([]byte("another"), ms.Messages()[0].Key())
		req.Equal([]byte("one another"), ms.Messages()[0].Value())
		count++
	}
	req.Equal(1, count)

	scanner = commitlog.NewSegmentScanner(cleaned[1])
	count = 0
	for {
		ms, err = scanner.Scan()
		if err != nil {
			break
		}
		req.Equal(1, len(ms.Messages()))
		req.Equal([]byte("travisjeffery"), ms.Messages()[0].Key())
		req.Equal([]byte("two tj"), ms.Messages()[0].Value())
		count++
	}
	req.Equal(1, count)

}

func newMessageSet(offset uint64, pmsgs ...*protocol.Message) commitlog.MessageSet {
	cmsgs := make([]commitlog.Message, 0, len(pmsgs))
	for _, msg := range pmsgs {
		b, err := protocol.Encode(msg)
		if err != nil {
			panic(err)
		}
		cmsgs = append(cmsgs, b)
	}
	return commitlog.NewMessageSet(offset, cmsgs...)
}
