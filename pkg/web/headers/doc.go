// Package headers provides typed parsers and serializers for HTTP header values.
// Each header type has a corresponding Parse<Name> function and a String method,
// offering a round-trip-stable representation.
//
// All types are immutable value types. Parsing never panics; errors are returned
// for malformed input. The zero value of each type is the valid "not present" state.
package headers
