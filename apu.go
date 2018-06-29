package vcsgo

// technically part of TIA, but w/e

type apu struct {
	// not marshalled in snapshot
	buffer apuCircleBuf

	FreqClk int

	SampleSum      int
	SampleSumCount int

	// everything else marshalled
	Channel0 sound
	Channel1 sound
}

func (apu *apu) init() {
	apu.Channel0.init()
	apu.Channel1.init()
}

type sound struct {
	Volume  byte
	FreqDiv byte
	Control byte

	FreqDivCounter    divCounter
	SubFreqDivCounter divCounter
	Div31Counter      divCounter
	PolyCounter4      int
	PolyCounter5      int
	PolyCounter9      int

	Out int
}

func (s *sound) init() {
	s.PolyCounter4 = 0xffff
	s.PolyCounter5 = 0xffff
	s.PolyCounter9 = 0xffff
}

type divCounter struct {
	Counter byte
}

func (d *divCounter) tick(val byte) bool {
	d.Counter++
	if d.Counter < val {
		return false
	}
	d.Counter = 0
	return true
}

func (s *sound) runFreqCycle() {
	if !s.FreqDivCounter.tick(s.FreqDiv) {
		return
	}

	switch s.Control {
	case 0x0, 0xb:
		s.Out = 1
	case 0x1:
		s.Out = s.run4BitPoly()
	case 0x2:
		if s.SubFreqDivCounter.tick(15) {
			s.Out = s.run4BitPoly()
		}
	case 0x3:
		if s.run5BitPoly() == 1 {
			s.Out = s.run4BitPoly()
		}
	case 0x4, 0x5:
		s.Out ^= 1
	case 0x6, 0xa: // div 31
		s.Out = s.runDiv31()
	case 0x7, 0x9:
		s.Out = s.run5BitPoly()
	case 0x8:
		s.Out = s.run9BitPoly()
	case 0xc, 0xd:
		if s.SubFreqDivCounter.tick(3) {
			s.Out ^= 1
		}
	case 0xe:
		if s.SubFreqDivCounter.tick(3) {
			s.Out = s.runDiv31()
		}
	case 0xf:
		if s.SubFreqDivCounter.tick(3) {
			s.Out = s.run5BitPoly()
		}
	}
}

func (s *sound) runDiv31() int {
	if s.Out == 1 {
		if s.Div31Counter.tick(18) {
			return 0
		}
	} else {
		if s.Div31Counter.tick(13) {
			return 1
		}
	}
	return s.Out
}

func (s *sound) run4BitPoly() int {
	out := s.PolyCounter4 & 1
	s.PolyCounter4 >>= 1
	s.PolyCounter4 &^= 0x08
	in := s.PolyCounter4 & 1
	s.PolyCounter4 |= (out ^ in) << 3
	return out
}

func (s *sound) run5BitPoly() int {
	out := s.PolyCounter5 & 1
	s.PolyCounter5 >>= 1
	s.PolyCounter5 &^= 0x10
	in := (s.PolyCounter5 >> 1) & 1
	s.PolyCounter5 |= (out ^ in) << 4
	return out
}

func (s *sound) run9BitPoly() int {
	out := s.PolyCounter9 & 1
	s.PolyCounter9 >>= 1
	s.PolyCounter9 &^= 0x100
	in := (s.PolyCounter9 >> 3) & 1
	s.PolyCounter9 |= (out ^ in) << 8
	return out
}

const (
	amountGenerateAhead = 32 * 4
	samplesPerSecond    = 44100
	timePerSample       = 1.0 / samplesPerSecond
)

const apuCircleBufSize = amountGenerateAhead

const clocksPerSecond = 3 * 1.19 * 1000 * 1000
const clocksPerSample = clocksPerSecond / samplesPerSecond

const timePerCycle = 1.0 / clocksPerSecond

// NOTE: size must be power of 2
type apuCircleBuf struct {
	writeIndex uint
	readIndex  uint
	buf        [apuCircleBufSize]byte
}

func (c *apuCircleBuf) write(bytes []byte) (writeCount int) {
	for _, b := range bytes {
		if c.full() {
			return writeCount
		}
		c.buf[c.mask(c.writeIndex)] = b
		c.writeIndex++
		writeCount++
	}
	return writeCount
}
func (c *apuCircleBuf) read(preSizedBuf []byte) []byte {
	readCount := 0
	for i := range preSizedBuf {
		if c.size() == 0 {
			break
		}
		preSizedBuf[i] = c.buf[c.mask(c.readIndex)]
		c.readIndex++
		readCount++
	}
	return preSizedBuf[:readCount]
}
func (c *apuCircleBuf) mask(i uint) uint { return i & (uint(len(c.buf)) - 1) }
func (c *apuCircleBuf) size() uint       { return c.writeIndex - c.readIndex }
func (c *apuCircleBuf) full() bool       { return c.size() == uint(len(c.buf)) }

func (apu *apu) runCycle() {

	if !apu.buffer.full() {

		apu.FreqClk++
		if apu.FreqClk == 114 {
			apu.runFreqCycle()
			apu.FreqClk = 0
		}

		c0 := apu.Channel0.Out * int(apu.Channel0.Volume)
		c1 := apu.Channel1.Out * int(apu.Channel1.Volume)

		apu.SampleSum += c0 + c1
		apu.SampleSumCount++
		countLimit := clocksPerSample // for trunc
		if apu.SampleSumCount >= int(countLimit) {

			sum := float32(apu.SampleSum) / 30.0 // 2 channels, 15 vol levels

			output := sum / float32(apu.SampleSumCount)

			apu.SampleSum = 0
			apu.SampleSumCount = 0

			sample := int16(output * 32767.0)
			apu.buffer.write([]byte{
				byte(sample & 0xff),
				byte(sample >> 8),
				byte(sample & 0xff),
				byte(sample >> 8),
			})
		}
	}
}

func (apu *apu) runFreqCycle() {
	apu.Channel0.runFreqCycle()
	apu.Channel1.runFreqCycle()
}
