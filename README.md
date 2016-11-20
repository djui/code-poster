# code-poster

Produce an image poster from a path of source files.

To get inspired and see some examples, visit: https://commits.io


## Installation

    go get -u github.com/djui/code-poster


## Usage

    code-poster -h
    Usage of code-poster:
      -alphathreshold float
        	alpha threshold for image mask [%] (default 0.5)
      -bgcolor string
        	foreground/text color [rgba] (default "#ffffff")
      -debug
        	debug mode
      -fgcolor string
        	foreground/text color [rgba] (default "#808080")
      -fontname string
        	font name (default "monospace")
      -fontsize float
        	font size [cm] (default 0.1)
      -fontspacing float
        	font glyph spacing [%]
      -height float
        	height [cm] (default 12)
      -mask string
        	path to image mask [png]
      -maskdpi int
        	DPI of image mask (default 72)
      -maskscale float
        	scale of image mask [%] (default 1)
      -monochrome
        	monochrome mode
      -out string
        	path to image output [svg]
      -src string
        	path to source code
      -srcpattern string
        	pattner of source code filename (default "*.go")
      -width float
        	width [cm] (default 10)


## Examples

    code-poster -srcpattern "*.py" -out poster.svg -mask mask.png
    inkscape --without-gui "$PWD/poster.svg" -A "$PWD/poster.pdf"


## Restrictions

- The output file must be of filetype SVG
- The mask file must be of filetype PNG
