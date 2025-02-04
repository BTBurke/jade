// Code generated by "jade.go"; DO NOT EDIT.

package imp

import (
	"io"

	"github.com/Joker/hpp"
	"github.com/Joker/jade/testdata/imp/model"
)

const (
	simple__0 = `<html><body><h1>`
	simple__1 = `</h1><p>Here's a list of your favorite colors:</p><ul>`
	simple__2 = `</ul><ul>`
	simple__3 = `</ul></body></html>`
	simple__4 = `<li>`
	simple__5 = `</li>`
	simple__7 = `</li><li>`
)

func Simple(u *model.User, st []model.Story, wr io.Writer) {

	r, w := io.Pipe()
	go func() {
		buffer := &WriterAsBuffer{w}

		buffer.WriteString(simple__0)
		WriteEscString(u.FirstName, buffer)
		buffer.WriteString(simple__1)

		for _, colorName := range u.FavoriteColors {
			buffer.WriteString(simple__4)
			WriteEscString(colorName, buffer)
			buffer.WriteString(simple__5)
		}
		buffer.WriteString(simple__2)

		for _, story := range st {
			buffer.WriteString(simple__4)
			WriteInt(int64(story.StoryId), buffer)
			buffer.WriteString(simple__7)
			WriteInt(int64(story.UserId), buffer)
			buffer.WriteString(simple__7)
			WriteEscString(story.UserName, buffer)
			buffer.WriteString(simple__5)
		}
		buffer.WriteString(simple__3)

		w.Close()
	}()
	hpp.Format(r, wr)
}
