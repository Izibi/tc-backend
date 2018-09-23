
package model

type Facets struct {
  Base bool
  Admin bool
  Member bool
}

var NullFacet Facets = Facets{}
var BaseFacet Facets = Facets{Base: true}
