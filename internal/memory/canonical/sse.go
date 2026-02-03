package canonical

import (
	"bytes"
	"strings"
)

type streamSpan struct {
	Start int
	End   int
}

type sseData struct {
	Data string
	Span streamSpan
}

type sseLine struct {
	data string
	span streamSpan
}

type sseParser struct {
	buffer       []byte
	bufferOffset int
	eventLines   []sseLine
}

func (p *sseParser) Feed(chunk []byte, chunkOffset int) []sseData {
	if len(chunk) == 0 {
		return nil
	}
	if len(p.buffer) == 0 {
		p.bufferOffset = chunkOffset
	}
	p.buffer = append(p.buffer, chunk...)

	var out []sseData
	for {
		idx := bytes.IndexByte(p.buffer, '\n')
		if idx < 0 {
			break
		}
		lineBytes := p.buffer[:idx]
		lineStart := p.bufferOffset
		lineEnd := p.bufferOffset + idx

		p.buffer = p.buffer[idx+1:]
		p.bufferOffset += idx + 1

		if len(lineBytes) > 0 && lineBytes[len(lineBytes)-1] == '\r' {
			lineBytes = lineBytes[:len(lineBytes)-1]
			lineEnd--
		}

		out = append(out, p.consumeLine(lineBytes, lineStart, lineEnd)...)
	}
	return out
}

func (p *sseParser) Flush() []sseData {
	var out []sseData
	if len(p.buffer) > 0 {
		lineStart := p.bufferOffset
		lineEnd := p.bufferOffset + len(p.buffer)
		lineBytes := p.buffer
		if len(lineBytes) > 0 && lineBytes[len(lineBytes)-1] == '\r' {
			lineBytes = lineBytes[:len(lineBytes)-1]
			lineEnd--
		}
		out = append(out, p.consumeLine(lineBytes, lineStart, lineEnd)...)
		p.buffer = nil
	}
	out = append(out, p.flushEvent()...)
	return out
}

func (p *sseParser) consumeLine(line []byte, lineStart int, lineEnd int) []sseData {
	if len(line) == 0 {
		return p.flushEvent()
	}
	text := string(line)
	if strings.HasPrefix(text, "data:") {
		dataStart := lineStart + len("data:")
		if len(line) > len("data:") && line[len("data:")] == ' ' {
			dataStart++
		}
		dataEnd := lineEnd
		data := strings.TrimLeft(text[len("data:"):], " ")
		p.eventLines = append(p.eventLines, sseLine{
			data: data,
			span: streamSpan{Start: dataStart, End: dataEnd},
		})
	}
	return nil
}

func (p *sseParser) flushEvent() []sseData {
	if len(p.eventLines) == 0 {
		return nil
	}
	out := make([]sseData, 0, len(p.eventLines))
	for _, line := range p.eventLines {
		out = append(out, sseData{
			Data: line.data,
			Span: line.span,
		})
	}
	p.eventLines = nil
	return out
}
