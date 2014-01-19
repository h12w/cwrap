Cwrap: Wraps C libraries in Go
==============================

Cwrap is a Go wrapper generator for C libraries.

Features
--------
* No Cgo types exposed out of the wrapper package, and uses as less allocation/copy as possible.
* C name prefix mapped to Go packages, and a wrapper package can import another wrapper package.
* Follows Go naming conventions.
* C union.
* Use Go language features when possible:
  * string and bool.
  * Multiple return values.
  * Slice, slice of slice and slice of string.
  * struct with methods. 
  * Go closures as callbacks.
* Stay out of the way when you need to do it manually for specified declarations.

Usage
-----
Cwrap itself is a Go package rather than an executable program. Just fill a cwrap.Package struct literal and call it's Wrap method to generate your wrapper package under $GOPATH. Here is a simple example:

Say you want to generate a wrapper package for SDL2, and its header is at

    /usr/local/include/SDL2/SDL2.h

So the cwrap.Package literal looks like:

    var sdl = &Package{
		PacName: "sdl",
		PacPath: "go-sdl",
		From: Header{
			Dir:           "/usr/local/include/",
			File:          "SDL2/SDL.h",
			OtherCode:     "#define _SDL_main_h",
			NamePattern:   `\ASDL(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{"pkg-config: sdl2"},
			BoolTypes:     []string{"SDL_bool"},
		},
		Included: []*Package{},
	}

Then just call

    err := sdl.Wrap()

Examples
--------
In the examples directory, there are C libraries that I have successfully applied Cwrap, including:
* Cairo
* GSL (GNU Scientific Library)
* MuPDF
* PLplot
* SDL2 (Simple DirectMedia Layer)

You are very welcome to submit examples you think useful to others.

Applications
------------
* gr: A minimal PDF viewer based on SDL2 and MuPDF (https://github.com/hailiang/gr)

Issue Report
------------
Cwrap may not cover every possible case and fails to come up with a corrresonding Go type or convertion, then the generated code may not be able to compile. When this happens, do the following steps:

1. Comment out the failed function wrappers till it compiles.
2. Add the C names of these failed functions to the excluded list (Package.From.Excluded).
3. Submit the generator example to me. I cannot guarantee anything but I will try to fix critical issues.

TODO
----
* Go idiomatic error handling (return error for each function/method).
* Godoc documentation.
* Alignment and padding of generated Go struct fields may need more careful checking (It just works fine for now, and I won't spend time on this until a real bug is found).

Limitations
-----------
* C variadic functions (...) are not supported.

Acknowledgement
---------------
Cwrap uses gccxml (http://gccxml.github.io) to parse C headers to an XML file. Thanks very much for their excellent work.
