package tui

import tea "github.com/charmbracelet/bubbletea"

type liveOutputMsg struct {
	stream string
	data   string
}

type liveOutputDoneMsg struct{}

type liveOutputWriter struct {
	ch     chan<- liveOutputMsg
	stream string
}

func (w liveOutputWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if w.ch == nil {
		return len(p), nil
	}
	data := string(append([]byte{}, p...))
	w.ch <- liveOutputMsg{stream: w.stream, data: data}
	return len(p), nil
}

func listenLiveOutputCmd(ch <-chan liveOutputMsg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return liveOutputDoneMsg{}
		}
		return msg
	}
}
