# contextlock

[godocs](https://pkg.go.dev/github.com/sakjur/contextlock)

_contextlock_ is a small library for putting values in a Go 
[context.Context](https://pkg.go.dev/context#Context) that should not be
immediately accessible. The locks placed on the context value could be
used by a web application to assert that authentication is followed by
proper authorization checks before some value is used.

The library is _feature-complete_ as-is and is unlikely to get any
further lock types beyond the boolean, function, and time locks already
provided. Many applications might consider already this limited set of
locks to be exaggerated, and I encourage you to copy over the code that
you need to your codebase to remove the dependency on code you don't.

At the moment, the library is missing documentation and examples.

Licensed under the [MIT No Attribution](https://github.com/aws/mit-0)
license.
