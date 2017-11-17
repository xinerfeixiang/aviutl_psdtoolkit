package ipc

import (
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/oov/aviutl_psdtoolkit/src/go/img"
	"github.com/oov/aviutl_psdtoolkit/src/go/imgmgr/editing"
	"github.com/oov/aviutl_psdtoolkit/src/go/imgmgr/source"
	"github.com/oov/aviutl_psdtoolkit/src/go/imgmgr/temporary"
	"github.com/oov/aviutl_psdtoolkit/src/go/ods"
	"github.com/oov/psd/blend"
)

type IPC struct {
	ShowGUI      func() (uintptr, error)
	Deserialized func()

	ImageSrcs    source.Sources
	EditingImg   editing.Editing
	TemporaryImg temporary.Temporary

	queue     chan func()
	reply     chan error
	replyDone chan struct{}
}

func (ipc *IPC) do(f func()) {
	done := make(chan struct{})
	ipc.queue <- func() {
		f()
		done <- struct{}{}
	}
	<-done
}

func (ipc *IPC) load(id int, filePath string) (*img.Image, error) {
	return ipc.TemporaryImg.Load(id, filePath)
}

func (ipc *IPC) draw(id int, filePath string, width, height int) ([]byte, error) {
	img, err := ipc.TemporaryImg.Load(id, filePath)
	if err != nil {
		return nil, errors.Wrap(err, "ipc: could not load")
	}
	startAt := time.Now().UnixNano()
	rgba, err := img.Render(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "ipc: could not render")
	}
	ret := image.NewRGBA(image.Rect(0, 0, width, height))
	blend.Copy.Draw(ret, ret.Rect, rgba, image.Pt(int(float32(-img.OffsetX)*img.Scale), int(float32(-img.OffsetY)*img.Scale)))
	rgbaToNBGRA(ret.Pix)
	ipc.ImageSrcs.Logger.Println(fmt.Sprintf("render: %dms", (time.Now().UnixNano()-startAt)/1e6))
	return ret.Pix, nil
}

func (ipc *IPC) getLayerNames(id int, filePath string) (string, error) {
	img, err := ipc.TemporaryImg.Load(id, filePath)
	if err != nil {
		return "", errors.Wrap(err, "ipc: could not load")
	}
	s := make([]string, len(img.Layers.Flat))
	for path, index := range img.Layers.FullPath {
		s[index] = path
	}
	return strings.Join(s, "\n"), nil
}

func (ipc *IPC) setProps(id int, filePath string, layer *string, scale *float32, offsetX, offsetY *int) (bool, int, int, error) {
	img, err := ipc.TemporaryImg.Load(id, filePath)
	if err != nil {
		return false, 0, 0, errors.Wrap(err, "ipc: could not load")
	}
	modified := img.Modified
	if layer != nil {
		if *layer != "" {
			l := *img.InitialLayerState + " " + *layer
			layer = &l
		} else {
			layer = img.InitialLayerState
		}
		b, err := img.Deserialize(*layer)
		if err != nil {
			return false, 0, 0, errors.Wrap(err, "ipc: deserialize failed")
		}
		if b {
			modified = true
		}
	}
	if scale != nil {
		if *scale > 1 {
			*scale = 1
		} else if *scale < 0.00001 {
			*scale = 0.00001
		}
		if *scale != img.Scale {
			img.Scale = *scale
			modified = true
		}
	}
	if offsetX != nil {
		if *offsetX != img.OffsetX {
			img.OffsetX = *offsetX
			modified = true
		}
	}
	if offsetY != nil {
		if *offsetY != img.OffsetY {
			img.OffsetY = *offsetY
			modified = true
		}
	}
	r := img.ScaledCanvasRect()
	img.Modified = modified
	return modified, r.Dx(), r.Dy(), nil
}

func (ipc *IPC) showGUI() (uintptr, error) {
	h, err := ipc.ShowGUI()
	if err != nil {
		return 0, errors.Wrap(err, "ipc: cannot show the gui")
	}
	return h, nil
}

func (ipc *IPC) serialize() (string, error) {
	return ipc.EditingImg.Serialize()
}

func (ipc *IPC) deserialize(state string) error {
	if err := ipc.EditingImg.Deserialize(state); err != nil {
		return err
	}
	ipc.Deserialized()
	return nil
}

func (ipc *IPC) SendEditingImageState(filePath, state string) error {
	var err error
	ipc.do(func() {
		fmt.Print("EDIS")
		ods.ODS("  FilePath: %s", filePath)
		if err = writeString(filePath); err != nil {
			return
		}
		ods.ODS("  State: %s", state)
		if err = writeString(state); err != nil {
			return
		}
	})
	if err != nil {
		return err
	}
	ods.ODS("wait EDIS reply...")
	err = <-ipc.reply
	ods.ODS("wait EDIS reply ok")
	ipc.replyDone <- struct{}{}
	return err
}

func (ipc *IPC) CopyFaviewValue(filePath, sliderName, name, value string) error {
	var err error
	ipc.do(func() {
		fmt.Print("CPFV")
		ods.ODS("  FilePath: %s", filePath)
		if err = writeString(filePath); err != nil {
			return
		}
		ods.ODS("  SliderName: %s / Name: %s / Value: %s", sliderName, name, value)
		if err = writeString(sliderName); err != nil {
			return
		}
		if err = writeString(name); err != nil {
			return
		}
		if err = writeString(value); err != nil {
			return
		}
	})
	if err != nil {
		return err
	}
	ods.ODS("wait CPFV reply...")
	err = <-ipc.reply
	ods.ODS("wait CPFV reply ok")
	ipc.replyDone <- struct{}{}
	return err
}

func (ipc *IPC) ExportFaviewSlider(filePath, sliderName string, names, values []string) error {
	var err error
	ipc.do(func() {
		fmt.Print("EXFS")
		ods.ODS("  FilePath: %s", filePath)
		if err = writeString(filePath); err != nil {
			return
		}
		ods.ODS("  SliderName: %s / Names: %v / Values: %v", sliderName, names, values)
		if err = writeString(sliderName); err != nil {
			return
		}
		if err = writeString(strings.Join(names, "\x00")); err != nil {
			return
		}
		if err = writeString(strings.Join(values, "\x00")); err != nil {
			return
		}
	})
	if err != nil {
		return err
	}
	ods.ODS("wait EXFS reply...")
	err = <-ipc.reply
	ods.ODS("wait EXFS reply ok")
	ipc.replyDone <- struct{}{}
	return err
}

func (ipc *IPC) dispatch(cmd string) error {
	switch cmd {
	case "HELO":
		return writeUint32(0x80000000)

	case "DRAW":
		id, filePath, err := readIDAndFilePath()
		if err != nil {
			return err
		}
		width, err := readInt32()
		if err != nil {
			return err
		}
		height, err := readInt32()
		if err != nil {
			return err
		}
		ods.ODS("  Width: %d / Height: %d", width, height)
		b, err := ipc.draw(id, filePath, width, height)
		if err != nil {
			return err
		}
		if err = writeUint32(0x80000000); err != nil {
			return err
		}
		return writeBinary(b)

	case "LNAM":
		id, filePath, err := readIDAndFilePath()
		if err != nil {
			return err
		}
		s, err := ipc.getLayerNames(id, filePath)
		if err != nil {
			return err
		}
		if err = writeUint32(0x80000000); err != nil {
			return err
		}
		return writeString(s)

	case "PROP":
		id, filePath, err := readIDAndFilePath()
		if err != nil {
			return err
		}
		const (
			propEnd = iota
			propLayer
			propScale
			propOffsetX
			propOffsetY
		)
		var layer *string
		var scale *float32
		var offsetX, offsetY *int
	readProps:
		for {
			pid, err := readInt32()
			if err != nil {
				return err
			}
			switch pid {
			case propEnd:
				break readProps
			case propLayer:
				s, err := readString()
				if err != nil {
					return err
				}
				layer = &s
				ods.ODS("  Layer: %s", s)
			case propScale:
				f, err := readFloat32()
				if err != nil {
					return err
				}
				scale = &f
				ods.ODS("  Scale: %f", f)
			case propOffsetX:
				i, err := readInt32()
				if err != nil {
					return err
				}
				offsetX = &i
				ods.ODS("  OffsetX: %d", i)
			case propOffsetY:
				i, err := readInt32()
				if err != nil {
					return err
				}
				offsetY = &i
				ods.ODS("  OffsetY: %d", i)
			}
		}
		modified, width, height, err := ipc.setProps(id, filePath, layer, scale, offsetX, offsetY)
		if err != nil {
			return err
		}
		ods.ODS("  Modified: %v / Width: %d / Height: %d", modified, width, height)
		if err = writeUint32(0x80000000); err != nil {
			return err
		}
		if err = writeBool(modified); err != nil {
			return err
		}
		if err = writeUint32(uint32(width)); err != nil {
			return err
		}
		return writeUint32(uint32(height))

	case "SGUI":
		h, err := ipc.showGUI()
		if err != nil {
			return err
		}
		if err = writeUint32(0x80000000); err != nil {
			return err
		}
		return writeUint64(uint64(h))

	case "SRLZ":
		s, err := ipc.serialize()
		if err != nil {
			return err
		}
		if err = writeUint32(0x80000000); err != nil {
			return err
		}
		return writeString(s)

	case "DSLZ":
		s, err := readString()
		if err != nil {
			return err
		}
		err = ipc.deserialize(s)
		if err != nil {
			return err
		}
		if err = writeUint32(0x80000000); err != nil {
			return err
		}
		return writeBool(true)
	}
	return errors.New("unknown command")
}

type odsLogger struct{}

func (odsLogger) Println(v ...interface{}) {
	ods.ODS(fmt.Sprintln(v...))
}

func (ipc *IPC) Init() {
	ipc.ImageSrcs.Logger = odsLogger{}
	ipc.EditingImg.Srcs = &ipc.ImageSrcs
	ipc.TemporaryImg.Srcs = &ipc.ImageSrcs

	ipc.queue = make(chan func())
	ipc.reply = make(chan error)
	ipc.replyDone = make(chan struct{})
}

func (ipc *IPC) readCommand(r chan string) {
	cmd := make([]byte, 4)
	for {
		ods.ODS("wait next command...")
		if read, err := io.ReadFull(os.Stdin, cmd); err != nil || read != 4 {
			r <- fmt.Sprintf("error: %v", err)
			return
		}
		l := binary.LittleEndian.Uint32(cmd)
		if l&0x80000000 == 0 {
			break
		}
		l &= 0x7fffffff
		if l == 0 {
			ods.ODS("readCommand: reply no error")
			ipc.reply <- nil
		} else {
			buf := make([]byte, l)
			read, err := io.ReadFull(os.Stdin, buf)
			if err != nil {
				r <- fmt.Sprintf("error: %v", err)
				return
			}
			if read != int(l) {
				r <- fmt.Sprintf("error: %v", errors.New("unexcepted read size"))
				return
			}
			ods.ODS("readCommand: reply %s", string(buf))
			ipc.reply <- errors.New(string(buf))
		}
		<-ipc.replyDone
	}
	ods.ODS("readCommand: cmd %s", string(cmd))
	r <- string(cmd)
}

func (ipc *IPC) Main(exitCh chan<- struct{}) {
	gcTicker := time.NewTicker(1 * time.Minute)
	defer func() {
		if err := recover(); err != nil {
			ods.Recover(err)
		}
		gcTicker.Stop()
		close(exitCh)
	}()

	cmdCh := make(chan string)
	go ipc.readCommand(cmdCh)
	for {
		select {
		case <-gcTicker.C:
			ipc.EditingImg.Touch()
			ipc.TemporaryImg.GC()
			ipc.ImageSrcs.GC()
		case f := <-ipc.queue:
			f()
		case cmd := <-cmdCh:
			if len(cmd) != 4 {
				ods.ODS("%s", cmd) // error report
				return
			}
			ods.ODS("%s", cmd)
			if err := ipc.dispatch(cmd); err != nil {
				ods.ODS("error: %v", err)
				if err = writeReply(err); err != nil {
					return
				}
			}
			ods.ODS("%s END", cmd)
			go ipc.readCommand(cmdCh)
		}
	}
}
