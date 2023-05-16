package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"

	"fyne.io/fyne/v2/widget"
)

func convertUnits(in string) int64 {
	number, err := strconv.ParseInt(strings.TrimRight(in, "KMG"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	unit := []byte(in)[len(in)-1]
	switch unit {
	case 'K':
		return number * 1024
	case 'M':
		return number * 1024 * 1024
	case 'G':
		return number * 1024 * 1024 * 1024
	default:
		return 0
	}
}
func main() {
	a := app.New()
	w := a.NewWindow("Brokedown's Flash Layout Tool")

	f := NewFlash()

	blocksContainer := container.New(layout.NewVBoxLayout())
	aboveBlocksForm := widget.NewForm()
	blocksForm := widget.NewForm()
	flashSizeEntry := widget.NewEntry()
	aboveBlocksForm.Append("Flash Size", flashSizeEntry)
	sizeSelect := widget.NewSelect([]string{"Custom", "256K", "512K", "1M", "2M", "4M", "8M", "16M", "32M", "64M", "128M", "256M", "512M", "1G"}, func(value string) {
		if value != "Custom" {
			flashSizeEntry.SetText(fmt.Sprintf("%d", convertUnits(value)))
			f.Size = int(convertUnits(value))
		}
	})
	aboveBlocksForm.Append("Quick Select", sizeSelect)
	automaticOffset := widget.NewCheck("Automatic Offset (in order starting at 0x0)", func(b bool) {
		f.AutomaticOffset = b

	})

	addButton := widget.NewButton("Add Block", func() {
		b := f.NewBlock()
		editBlock(b, f, blocksForm)
	})
	saveItem := widget.NewButton("Save", func() {
		dialog.NewFileSave(func(u fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.NewError(err, w).Show()
				return
			}
			out, locations, err := f.Assemble()
			if err != nil {
				dialog.NewError(err, w).Show()
				return
			}
			log.Printf("Writing %d bytes", len(out))
			u.Write(out)
			err = os.WriteFile(u.URI().Path()+".txt", locations, 0777)
			if err != nil {
				dialog.NewError(fmt.Errorf("location write error. Locations are: %s", locations), w).Show()
			}
			u.Close()
			dialog.NewInformation("Done", "Write complete", w).Show()
		}, w).Show()
	})
	blocksContainer.Add(automaticOffset)
	blocksContainer.Add(aboveBlocksForm)
	blocksContainer.Add(blocksForm)
	blocksContainer.Add(addButton)
	blocksContainer.Add(saveItem)

	w.SetContent(blocksContainer)
	w.Resize(fyne.Size{Width: 400, Height: 600})
	w.ShowAndRun()
}

type blockFormItems struct {
	bw          fyne.Window
	blockForm   *widget.Form
	blockOffset *widget.Entry
	padToSize   *widget.Entry
	padWithData *widget.Entry
}

func editBlock(b *FlashBlock, f *Flash, blocksForm *widget.Form) {
	var bfi blockFormItems
	bfi.bw = fyne.CurrentApp().NewWindow("Block")
	bfi.blockForm = widget.NewForm()
	bfi.blockOffset = widget.NewEntry()
	bfi.blockOffset.SetText(fmt.Sprintf("0x%x", b.Offset))
	if !f.AutomaticOffset {
		bfi.blockForm.Append("Offset", bfi.blockOffset)
	}
	bfi.padToSize = widget.NewEntry()
	bfi.padWithData = widget.NewEntry()
	bfi.padToSize.SetText(fmt.Sprintf("0x%x", b.PadToSize))
	bfi.padWithData.SetText(fmt.Sprintf("0x%x", b.PadWithData))

	fileLabel := widget.NewLabel(b.Filename)
	bfi.blockForm.Append("File", fileLabel)
	blockFileButton := widget.NewButton("Select File", func() {
		d := dialog.NewFileOpen(func(u fyne.URIReadCloser, err error) {
			if u == nil || err != nil {
				log.Println("Cancel")
				// Cancel button
				return
			}
			log.Printf("%#V", u)
			log.Printf("%s", u.URI().Path())
			fileLabel.SetText(path.Base(u.URI().Path()))
			buf, err := os.ReadFile(u.URI().Path())
			if err != nil {
				log.Println(err)
				dialog.NewError(err, bfi.bw).Show()
				return
			}
			log.Printf("Read %d bytes", len(buf))
			b.Data = buf
			b.Filename = fileLabel.Text
			bfi.padToSize.SetText(fmt.Sprintf("0x%x", len(buf)))
			log.Printf("filename is %s", b.Filename)
		}, bfi.bw)
		d.Show()
	})
	bfi.blockForm.AppendItem(widget.NewFormItem("Pad to size", bfi.padToSize))
	bfi.blockForm.AppendItem(widget.NewFormItem("Padding value", bfi.padWithData))

	bfi.blockForm.Append("Select File", blockFileButton)

	submitButton := widget.NewButton("Submit", func() {
		if !validateBlock(b, bfi) {
			return
		}
		refreshBlockList(f, blocksForm)

		bfi.bw.Close()
	})
	bfi.blockForm.Append("Submit Block", submitButton)
	bfi.blockForm.Append("Delete Block", widget.NewButton("Delete Block", func() {
		f.DeleteBlock(b)
		refreshBlockList(f, blocksForm)
		bfi.bw.Close()
	}))
	bfi.bw.Resize(fyne.Size{Width: 640, Height: 480})
	bfi.bw.SetContent(bfi.blockForm)
	bfi.bw.Show()
}

func refreshBlockList(f *Flash, blocksForm *widget.Form) {
	log.Println("Refreshing block list")
	o := widget.NewForm()
	f.Sort()
	for x := range f.Blocks {
		b := f.Blocks[x]
		log.Printf("Adding block %d", b.Number)
		blocksItem := widget.NewFormItem(b.Filename, widget.NewButton("Edit", func() {
			editBlock(b, f, blocksForm)
			refreshBlockList(f, blocksForm)
		}))
		if !f.AutomaticOffset {
			blocksItem.Text=fmt.Sprintf("%s@0x%x", b.Filename, b.Offset)
		}
		o.AppendItem(blocksItem)
		o.Append("---", widget.NewSeparator())

	}
	blocksForm.Items = o.Items
	blocksForm.Refresh()
}

func validateBlock(b *FlashBlock, bfi blockFormItems) bool {
	if len(b.Filename) == 0 {
		dialog.NewError(errors.New("no file selected"), bfi.bw).Show()
		return false
	}
	// Convert offset hex value to
	tmp, err := strconv.ParseInt(strings.Replace(bfi.blockOffset.Text, "0x", "", -1), 16, 32)
	if err != nil {
		dialog.NewError(err, bfi.bw).Show()
		return false
	}
	s := bfi.padToSize.Text
	i, err := strconv.ParseInt(strings.Replace(s, "0x", "", -1), 16, 64)
	if err != nil {
		dialog.NewError(fmt.Errorf("padded size is not a valid hexadecimal number"), bfi.bw).Show()
		return false
	}
	if int(i) < len(b.Data) {
		dialog.NewError(fmt.Errorf("padded size must not be smaller than actual size"), bfi.bw).Show()
		bfi.padToSize.SetText(fmt.Sprintf("0x%x", len(b.Data)))
		return false
	}
	b.PadToSize = int(i)
	i, err = strconv.ParseInt(strings.Replace(bfi.padWithData.Text, "0x", "", -1), 16, 64)
	if err != nil {
		dialog.NewError(fmt.Errorf("padded size is not a valid hexadecimal number"), bfi.bw).Show()
		return false
	}
	b.PadWithData = byte(i)

	b.Offset = int(tmp)
	return true
}
