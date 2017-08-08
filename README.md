# http3d

Reference implementation for HTTP third party copies based on
bearer tokens, specially for those based on
[Macaroons](https://research.google.com/pubs/pub41892.html).

Right now, the token negotiations is based on dCache's approach,
which has been implemented by DPM as well.

This approach is coupled to the fact we are dealing with Macaroons,
but in principle, as long as the token negotiation mechanism
is properly defined, it should be easily adaptable to any opaque
token.

Inside litmus, there are a set of tests which only make
sense for the Macaroon approach, as it will try to deserialize
the token, and change the caveats. This can't apply for opaque
tokens.
