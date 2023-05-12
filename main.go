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
	blocksForm := widget.NewForm()
	flashSizeEntry := widget.NewEntry()
	blocksForm.Append("Flash Size", flashSizeEntry)
	sizeSelect := widget.NewSelect([]string{"Custom", "256K", "512K", "1M", "2M", "4M", "8M", "16M", "32M", "64M", "128M", "256M", "512M", "1G"}, func(value string) {
		if value != "Custom" {
			flashSizeEntry.SetText(fmt.Sprintf("%d", convertUnits(value)))
			f.Size = int(convertUnits(value))
		}
	})
	blocksForm.Append("Quick Select", sizeSelect)
	automaticOffset := widget.NewCheck("Automatic Offset (in order starting at 0x0)", func(b bool) {
		f.AutomaticOffset = b

	})

	addButton := widget.NewButton("Add Block", func() {
		bw := a.NewWindow("Add Block")
		b := f.NewBlock()
		blockForm := widget.NewForm()
		blockOffset := widget.NewEntry()
		blockOffset.SetText("0x0")
		if !f.AutomaticOffset {
			blockForm.Append("Offset", blockOffset)
		}
		padToSize := widget.NewEntry()
		padWithData := widget.NewEntry()
		padToSize.SetText("0x0")
		padWithData.SetText("0xff")

		fileLabel := widget.NewLabel("...choose the file...")
		blockForm.Append("File", fileLabel)
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
					dialog.NewError(err, bw).Show()
					return
				}
				log.Printf("Read %d bytes", len(buf))
				b.Data = buf
				b.Filename = fileLabel.Text
				padToSize.SetText(fmt.Sprintf("0x%x", len(buf)))
				log.Printf("filename is %s", b.Filename)
			}, bw)
			d.Show()
		})
		blockForm.AppendItem(widget.NewFormItem("Pad to size", padToSize))
		blockForm.AppendItem(widget.NewFormItem("Padding value", padWithData))

		blockForm.Append("Select File", blockFileButton)

		submitButton := widget.NewButton("Submit", func() {
			if len(b.Filename) == 0 {
				dialog.NewError(errors.New("no file selected"), bw).Show()
				return
			}
			// Convert offset hex value to
			tmp, err := strconv.ParseInt(strings.Replace(blockOffset.Text, "0x", "", -1), 16, 32)
			if err != nil {
				dialog.NewError(err, w).Show()
				return
			}
			s := padToSize.Text
			i, err := strconv.ParseInt(strings.Replace(s, "0x", "", -1), 16, 64)
			if err != nil {
				dialog.NewError(fmt.Errorf("padded size is not a valid hexadecimal number"), bw).Show()
				return
			}
			if int(i) < len(b.Data) {
				dialog.NewError(fmt.Errorf("padded size must not be smaller than actual size"), bw).Show()
				padToSize.SetText(fmt.Sprintf("0x%x", len(b.Data)))
				return
			}
			b.PadToSize = int(i)
			i, err = strconv.ParseInt(strings.Replace(padWithData.Text, "0x", "", -1), 16, 64)
			if err != nil {
				dialog.NewError(fmt.Errorf("padded size is not a valid hexadecimal number"), bw).Show()
				return
			}
			b.PadWithData = byte(i)

			b.Offset = int(tmp)
			f.AddBlock(b)
			var blocksItem *widget.FormItem
			blocksItem = widget.NewFormItem(fmt.Sprintf("%s@%x", b.Filename, b.Offset), widget.NewButton("Delete", func() {
				log.Printf("Removing block %d", b.Number)
				f.DeleteBlock(b)
				for x := range blocksForm.Items {
					if blocksForm.Items[x] == blocksItem {
						blocksForm.Items = append(blocksForm.Items[0:x], blocksForm.Items[x+1:]...)
						break
					}
				}
				blocksForm.Refresh()

				blocksContainer.Refresh()
			}))
			blocksForm.AppendItem(blocksItem)
			bw.Close()
		})
		blockForm.Append("Submit Block", submitButton)
		blockForm.Append("", widget.NewSeparator())
		bw.Resize(fyne.Size{Width: 600, Height: 400})
		bw.SetContent(blockForm)
		bw.Show()
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
	blocksContainer.Add(blocksForm)
	blocksContainer.Add(addButton)
	blocksContainer.Add(saveItem)

	w.SetContent(blocksContainer)
	w.Resize(fyne.Size{Width: 400, Height: 600})
	w.ShowAndRun()
}
