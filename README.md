# leakybuffer
A leaky circuit breaker for your pipelines

## Installing
If you have the Go compiler installed and correctly set up:

    go get github.com/cafxx/leakybuffer

## Usage
Suppose you have a pipeline made of two processes connected by a UNIX pipe:

    $ a | b

If process `b` stops processing, or it is too slow, the pipe between `a` and `b`
will fill up. When this happens, writes in `a` will block. This is normally
desirable because it applies backpressure to the producer (`a`) and prevents the
amount of data in the pipe from growing indefinitely.

On the other hand, there are situations in which the backpressure is
undesirable, i.e. a slowdown or problem in `b` should not prevent `a` from
progressing. `leakybuffer` is designed to address this requirement.

    $ a | leakybuffer | b

Interposing `leakybuffer` between producer and consumer will make sure that
writes in `a` never block by consuming the pipe and forwarding the data to `b`.
If `b` stops reading or slows down, `leakybuffer` will drop data as needed to
keep `a` from blocking.

`leakybuffer` has an internal buffer (currently hardcoded to 2MB) that will be
used to avoid losing data as much as possible. Whenever `leakybuffer` needs to
drop data it will print a line to `stderr` containing the amount of data lost,
e.g.:

    warn: dropped 65536 bytes

Currently `leakybuffer` only reads from `stdin` and outputs to `stdout`. The
data is treated as a stream of bytes, no special care is given to newlines or
other separators.

## Performance
On my machine I ran a very unscientific test to understand how much overhead
`leakybuffer` adds:

- ```cat /dev/zero | pv -at >/dev/null``` 1.9GiB/s
- ```cat /dev/zero | tee | pv -at >/dev/null``` 1.37GiB/s
- ```cat /dev/zero | ./leakybuffer 2>/dev/null | pv -at >/dev/null``` 1.54GiB/s

## Notes
- `leakybuffer` is named after [`buffer(1)`][1]

[1]: http://linux.die.net/man/1/buffer
