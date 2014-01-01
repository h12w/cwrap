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
* Stay out of the way when you need to do it manually for specified definitions.

Examples
--------
In the examples directory, there are three libraries that I have successfully applied Cwrap: GNU Scientific Library, PLplot and Simple DirectMedia Layer.

TODO
----
* Alignment and padding of generated Go struct fields may need more careful checking (currently no checking is done at all, Cwrap just naively assume the same rules are applied to both Go and C).

Limitations
-----------
* C variadic functions (...) are not supported.

Acknowledgement
---------------
Cwrap uses gccxml (http://gccxml.github.io) to parse C headers to an XML file. Thanks very much for their excellent work.
