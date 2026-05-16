package chat

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mobiletoly/goldr/sse"
)

const EventsPath = "/chat/events"

func Events(w http.ResponseWriter, r *http.Request) {
	stream, ok := sse.Start(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	events, cancel := subscribe(r.Context(), eventAfterID(r))
	defer cancel()
	if err := stream.Comment("connected"); err != nil {
		return
	}
	stream.Flush()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if err := stream.Comment("keep-alive"); err != nil {
				return
			}
			stream.Flush()
		case message, ok := <-events:
			if !ok {
				return
			}
			if err := stream.Component(r, sse.ComponentEvent{
				ID:        strconv.FormatInt(message.ID, 10),
				Component: MessageView(message),
			}); err != nil {
				return
			}
			stream.Flush()
		}
	}
}

func eventsPath(afterID int64) string {
	if afterID <= 0 {
		return EventsPath
	}
	return EventsPath + "?after=" + strconv.FormatInt(afterID, 10)
}

func eventAfterID(r *http.Request) int64 {
	for _, value := range []string{sse.LastEventID(r), r.URL.Query().Get("after")} {
		id, err := strconv.ParseInt(value, 10, 64)
		if err == nil && id > 0 {
			return id
		}
	}
	return 0
}

func lastMessageID(messages []Message) int64 {
	if len(messages) == 0 {
		return 0
	}
	return messages[len(messages)-1].ID
}
