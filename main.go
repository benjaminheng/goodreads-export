package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/pelletier/go-toml"
)

var format = flag.String("format", "toml", "export format (options: toml)")

// GoodreadsBook is a representation of a row in the Goodreads exported CSV
// file.
type GoodreadsBook struct {
	Title     string `csv:"Title"`
	Author    string `csv:"Author"`
	DateRead  string `csv:"Date Read"`
	DateAdded string `csv:"Date Added"`
	Shelf     string `csv:"Exclusive Shelf"`
}

type date time.Time

func (d date) MarshalText() ([]byte, error) {
	t := time.Time(d)
	return []byte(t.Format("2006-05-04")), nil
}

// Book is our representation of a book
type Book struct {
	Title        string  `toml:"title"`
	Series       *string `toml:"series"`
	SeriesVolume *string `toml:"series_volume"`
	Author       string  `toml:"author"`
	ReadAt       *date   `toml:"read_at"`
	AddedAt      date    `toml:"added_at"`
	Shelf        string  `toml:"-"`
}

// FromGoodreadsBook transforms the Goodreads exported row into our
// representation.
func (b *Book) FromGoodreadsBook(src GoodreadsBook) (err error) {
	b.Author = src.Author
	b.Shelf = src.Shelf
	re := regexp.MustCompile(`(.+)\((.+), #(.+)\)$`)
	matches := re.FindStringSubmatch(src.Title)
	if len(matches) > 1 {
		b.Title = strings.TrimSpace(matches[1])
		b.Series = &matches[2]
		b.SeriesVolume = &matches[3]
	} else {
		b.Title = src.Title
	}
	if src.DateRead != "" {
		t, err := time.Parse("2006/05/04", src.DateRead)
		if err != nil {
			return err
		}
		d := date(t)
		b.ReadAt = &d
	}
	t, err := time.Parse("2006/05/04", src.DateAdded)
	if err != nil {
		return err
	}
	b.AddedAt = date(t)
	return nil
}

type Output struct {
	Read   []Book `toml:"read"`
	ToRead []Book `toml:"to_read"`
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	fmt.Println("  program [flags] <file>")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("file not provided")
	}
	fname := flag.Arg(0)
	f, err := os.OpenFile(fname, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	goodreadsBooks := []GoodreadsBook{}

	if err := gocsv.UnmarshalFile(f, &goodreadsBooks); err != nil {
		log.Fatal(err)
	}

	books := []Book{}
	for _, v := range goodreadsBooks {
		book := &Book{}
		if err := book.FromGoodreadsBook(v); err != nil {
			log.Fatal(err)
		}
		books = append(books, *book)
	}

	sort.Slice(books, func(i, j int) bool {
		if books[i].ReadAt == nil || books[j].ReadAt == nil {
			return true
		}
		return time.Time(*books[i].ReadAt).Before(time.Time(*books[j].ReadAt))
	})

	output := Output{}
	for _, v := range books {
		switch v.Shelf {
		case "read":
			output.Read = append(output.Read, v)
		case "to-read":
			output.ToRead = append(output.ToRead, v)
		}
	}

	switch *format {
	case "toml":
		if err := toml.NewEncoder(os.Stdout).Encode(output); err != nil {
			log.Fatal(err)
		}
	}
}
