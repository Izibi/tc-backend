
package model

type Facets struct {
  Base bool
  Admin bool
  Member bool
}

var BaseFacet Facets = Facets{Base: true}
