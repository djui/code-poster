all: build

build:
	go build

test: build
	./code-poster -src ~/dev/betalo/await -out poster-new.svg -mask mask-new.png -maskscale 2.0 -maskdpi 72  -fontspacing 0.2 -fgcolor "#dddddd"
	./code-poster -src ~/dev/betalo/await -out poster-old.svg -mask mask-old.png -maskscale 0.5 -maskdpi 144 -fontspacing 0.2 -fgcolor "#dddddd"

	inkscape --without-gui "$(CURDIR)/poster-new.svg" -A "$(CURDIR)/poster-new.pdf"
	inkscape --without-gui "$(CURDIR)/poster-old.svg" -A "$(CURDIR)/poster-old.pdf"
