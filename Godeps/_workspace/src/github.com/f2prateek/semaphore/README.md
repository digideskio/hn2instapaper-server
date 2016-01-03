# limiter

Easily ensure only a limited number of requests are run at a time.

# Usage
```
s := semaphore.New(5)

for {
    s.Acquire()
    go func() {
      defer s.Release()
      // do work; only 5 go-routines will be
      // executing simultaneously in this case.
    }()
}

```