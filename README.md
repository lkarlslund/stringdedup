# stringdedup - in memory string deduplication for Golang

Easy-peasy string deduplication to Golang. You can implement this in literally 2 minutes. But you might not want to - please read all of this.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/lkarlslund/stringdedup)](https://goreportcard.com/report/github.com/lkarlslund/stringdedup) 

## How

Instanciate a deduplication object by providing a function that takes a []byte slice and returns a hash. As stringdedup is using generics, your returned value type is user specified. For small amounts of strings an uint32 is fine, but you can use uint64 or even a [4]uint64 for a sha256 value. Choose something that is fast ;)

```
dedup := stringdedup.New(func(in []byte) uint32 {
	return xxhash.Checksum32(in)
})
```
Every time you encounter a string you want deduplicated, just wrap it in a deduplication call:

```
deduppedstring := dedup.S(inputstring)
```

You can also ingest []byte and get a deduplicated string back. This saves an allocation per call, se more detailed example below:

```
inputdata := []byte{0x01, 0x02, 0x03, 0x04}
deduppedstring := dedup.BS(inputdata)
```

## Why?

In a scenario where you read a lot of data containing repeated freeform strings, unless you do something special, you're wasting a lot of memory. A very simplistic example could be that you are indexing a lot of files - see the example folder in the package.

I use it in two different projects, and one of them gets a deduplication ratio of 1:5, saving a massive amount of memory. 

The example included shows that things are not black and white. You might gain something by using this package, and you might not. It really depends on what you are doing, and also how you are doing it.

## How do strings work, and why do I care? Isn't Go already clever when it comes to strings?

Yes, actually Go is quite clever about strings. Or at least as clever as technically possible, without it costing way too much CPU during normal scenarios.

Internally in Golang a string is defined as:

```
type string struct { // no, you can't do this in reality - use reflect.StringHeader
  p   pointer        // the bytes the string consists of, Golang internally uses "pointer" as the type, this is not a type reachable by mortals
  len int            // how many bytes are allocated
}
```
So the string variable takes up 12 bytes of space + the actual space the backing data use + some overhead for the heap management (?). This package tries to cut down on the duplicate backing data when your program introduces the same contents several times.

For the below explainations assume this is defined:
```
var a, b string
a = "ho ho ho"
```
### The rules of Go strings
- A string is a fixed length variable (!) with a pointer to variable length backing data (contents of the string)
- Strings are immutable (you can not change a string's backing data): `a = "no more" // one heap allocation, pointer and length is changed in a`
- Assigning a string to another string does not allocate new backing data: `b = a // pointer and length is changed in b (no heap allocation)`
- You can also cut up a string in smaller pieces without allocating new backing data (re-slicing): `b = a[0:5] // pointer and length is changed in b (no heap allocation)`
- Constant strings (hardcoded into your program) are still normal strings, the pointer does just not point to the heap but to your data section (?)

Every time you read data from somewhere external, run a string through a function (uppercase, lowercase etc), or convert from []byte to string, you allocate new backing data on the heap. 

## Okay, let's dedup!

Get the package:
```
go get github.com/lkarlslund/stringdedup
```

### Using stringdedup
- When you dedup something, and we don't know about it, it's *always* heap allocated and copied.
- If you have []byte, you can dedup it to a string in one call (reader.Read(b) -> stringdedup.BS(b))

```
dedup := stringdedup.New(func(in []byte) uint32 {
	return xxhash.Checksum32(in)
})
dedupedstring := dedup.S(somestring) // input string, get deduplicated string back
```

That's it! You're now guaranteed that this string only exists once in your program, if all the other string allocations process the same way.

If you're repeatedly reading from the same []byte buffer, you can save an allocation per call this way:
```
dedup := stringdedup.New(func(in []byte) uint32 {
	return xxhash.Checksum32(in)
})
buffer := make([]byte, 16384)
var mystrings []string
var err error
for err == nil {
  _, err = myreader.Read(buffer)
  // do some processing, oh you found something you want to save at buffer[42:103]
  mystrings = append(mystrings, dedup.BS(buffer[42:103])) // BS = input []byte, get deduplicated string back
} 
```
If you know that you're not going to dedup any of the existing strings in memory again, you can call:
```
  stringdedup.Flush()
```
This frees the hashing indexes from stringdedup. It doesn't mean you can not dedup again, it just means that stringdedup forgets about the strings that are already in memory.

## Caveats (there are some, and you better read this)

This package uses some tricks, that *may* break at any time, if the Golang developers choose to implement something differently. Namely it's using these particularities:

- Weak references by using a map of uintptr's
- Strings are removed from the deduplication map by using the SetFinalizer method. That means you can't use SetFinalizer on the strings that you put into or get back from the package. Golang really doesn't want you to use SetFinalizer, they see it as a horrible kludge, but I've found no other way of doing weak references with cleanup
- The strings are hash indexed via a 32-bit XXHASH. This is not a crypto safe hashing algorithm, but we're not doing this to prevent malicious collisions. This is about statistics, and I'd guess that you would have to store more than 400 million strings before you start to run into problems. Strings are validated before they're returned, so you will never get invalid data back. You could optimize this away if you're feeling really lucky.
- You can choose to purge the deduplication index by calling Flush() to free memory. New deduplicated strings start over, so now you might get duplicate strings anyway. Again, this is for specific scenarios.

This requires Go 1.18 on x86 / x64. Please let me know your experiences.

Twitter: @lkarlslund