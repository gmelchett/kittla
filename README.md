# The kittla programming language

A programming language with fearfull concurrency, eternal life times, blazing frequency, black traits, eel genetics,
laser blasters, eh what?? Naa, kittla is just another partial, probably buggy, hobby implementation something that looks
like Tcl.

I've known that there is programming language called Tcl for 20 years or so, but everyone that had an
opinion was kinda dismissive of Tcl. I recently read http://antirez.com/articoli/tclmisunderstood.html which surfaces
on Hacker News from time to time and to my surprised realized that the basic idea is beautiful
in its simplicity, yet effective, and still the code becomes readable. Not like some kind of rebus.
Since I've had an itch to write a lexer/parser/interpreter/vm for quite some time, I
couldn't resist the urge..


## Any use?

I plan to incorporate kittla into my console mpd client written in go that I've been hacking on from time to time for two-three
years by now. Not yet read for the public. I must bring it out of it's "eternal" beta stage
and smoke out some annoying bugs - not just add new features - like adding kittla :-)
So, no use.


## Why should I use it?

You shouldn't. Go away ;-) There are plenty of embedded languages for go that are far
more suitable for your hobby project. I'm perfectly happy with just one user of kittla - me :-)
The only cool thing about kittla is that is pretty small (currently < 1000 cloc) and easily extendable.


## Why the weird name "kittla"?

Since Tcl is pronounced "tickle" and "kittla" means "to tickle someone" in in Swedish, I though it would be suitable.

## Usage

You can embed kittla inside your own go program. Like this:
```
package main

import (
	"fmt"
	"kittla"
)

func main() {
	k := kittla.New()
	res, _, _ := k.Execute("set sum 0; set i 0; while {$i < 50} {inc i; set sum [eval $i+$sum]};")
	fmt.Println(string(res))
}

```

Or you can use  `kittlash` found in `cmd/kittlash`. Currently `kittlash` can only be used in
interactive mode.

## Language features

kittla is currently pretty much working like Tcl - with tons of features missing. Have a look
at https://en.wikipedia.org/wiki/Tcl#Syntax_and_fundamental_semantics

### Implemented features
  * Everything follows Tcl grammer, like: cmd args;
  * Pre evaluation with []
  * Post evaulation with {}
  * String ""
  * Escape codes like, \n etc
  * Comment with #
  * Long lines joined with \ as last char before new line

### Commands
  Currently using the Tcl naming, might change! (Some alias present)
  * break
  * continue
  * decr -- subtract value from variable
  * else
  * elseif
  * expr -- Calling "github.com/tidwall/expr" for a answer
  * if
  * incr -- increase variable with
  * puts -- print
  * set -- declare variable
  * unknown -- Called if command isn't known
  * while

Currently declaring own commands (or overloading) isn't supported.

## Future plans

Usually, I don't have plans for my hobby projects. I work on them as long as I find it fun.

## License
MIT
