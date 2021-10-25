package qparser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Resource determines the requested resource or the type of the resource
// ID might be empty string in case if a list of the resource type is requested
type Resource struct {
	Type string
	ID   string
}

// ResourceFields contains a list of requested fields for the resources
// 'fields[articles]=title,body' = ResourceFields{"articles": {"title", "body"}}
type ResourceFields map[string][]string

// FieldsByResource retrieves a list of fields by the given resource
// the second return value indicates if the list is set
func (r ResourceFields) FieldsByResource(resource string) (fields []string, ok bool) {
	if r == nil {
		return
	}
	fields, ok = r[resource]
	return
}

// Filter specifies field name to apply filtering to,
// a predicate expressed in textual form, the package does not know specific filtering syntax
// 'filter[createdAt]=lt:2015-01-01' = Filter{FieldName: "createdAt", Predicate: "lt:2015-01-01"}
type Filter struct {
	FieldName string
	Predicate string
}

// Include determines resources that should be included in a response
// 'include=comments.author' = Include{Relation: "comments", Includes: []Include{{Relation: "author"}}}
type Include struct {
	Relation string
	Includes []Include
}

// Page is pagination parameters
// 'page[size]=10&page[number]=2' = Page{Size: "10", Number: "2"}
// limit, offset, cursor are populated as well, the package is unaware of the pagination implementation
type Page struct {
	Size   string
	Number string
	Limit  string
	Offset string
	Cursor string
}

type SortOrder int

func (s SortOrder) String() string {
	if s == OrderDesc {
		return "DESC"
	}
	return "ASC"
}

const (
	OrderAsc SortOrder = iota
	OrderDesc
)

// Sort indicates the field by which the sorting should be performed and the sorting direction
type Sort struct {
	FieldName string
	Order     SortOrder
}

// Request represents the result of parsing the path and query string
type Request struct {
	Resource            Resource
	RelationshipType    string
	RelatedResourceType string
	Query               *Query
}

func (r *Request) IsRelationshipRequest() bool {
	return r.RelatedResourceType != ""
}

func (r *Request) IsRelatedResourceRequest() bool {
	return r.RelatedResourceType != ""
}

// Value represents the value from the query string
type Value struct {
	TopLevelKey string
	NestedKeys  []string
	Value       string
}

// Values maps a string top key to a list of values and nested keys.
type Values map[string][]Value

// Get gets the first value associated with the top key which contains all the nested keys.
// If there are no values associated with the combination, Get returns
// the empty string. To access multiple values, use the map directly.
func (v Values) Get(topKey string, nestedKeys ...string) string {
	if v == nil {
		return ""
	}
	list := v[topKey]
	if len(list) == 0 {
		return ""
	}

	for _, item := range list {
		if len(item.NestedKeys) != len(nestedKeys) {
			continue
		}
		if len(item.NestedKeys) == 0 && len(nestedKeys) == 0 {
			return item.Value
		}
		match := true
		for i, key := range nestedKeys {
			if item.NestedKeys[i] != key {
				match = false
				break
			}
		}
		if match {
			return item.Value
		}
	}
	return ""
}

// Query contains all parameters read from the query string
type Query struct {
	Includes []Include
	Fields   ResourceFields
	Sort     []Sort
	Filters  []Filter
	Page     *Page
	Values   Values
}

const (
	relationshipsRequest = "relationships"
)

// ParseValues parses a string and returns a structure filled with the corresponding values
// query string is expected to be a list of key=value settings separated by
// ampersands or semicolons. A setting without an equals sign is
// interpreted as a key set to an empty value.
// Query can contain nested keys, which are defined by square brackets,
// for example: page[size], page[number]
func ParseValues(query string) (Values, error) {
	var err error
	query, err = url.QueryUnescape(query)
	if err != nil {
		return nil, fmt.Errorf("qparser: failed to unescape query: %s", err.Error())
	}
	values := make(Values)
	if query != "" && query[0] == '?' {
		query = query[1:]
	}
	for query != "" {
		key := query
		if i := strings.IndexAny(key, "&;"); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}
		if key == "" {
			continue
		}
		value := ""
		if i := strings.Index(key, "="); i >= 0 {
			key, value = key[:i], key[i+1:]
		}
		topKey, nestedKeys := extractKeys(key)
		kv := Value{
			TopLevelKey: topKey,
			NestedKeys:  nestedKeys,
			Value:       value,
		}
		if _, ok := values[topKey]; !ok {
			values[topKey] = make([]Value, 0)
		}
		values[topKey] = append(values[topKey], kv)
	}
	return values, nil
}

// ParseQuery parses a string and returns a structure filled with the corresponding values
// Query is expected to be a list of key=value settings separated by
// ampersands or semicolons. A setting without an equals sign is
// interpreted as a key set to an empty value.
// Query can contain nested keys, which are defined by square brackets,
// for example: page[size], page[number]
func ParseQuery(query string) (*Query, error) {
	values, err := ParseValues(query)
	if err != nil {
		return nil, err
	}
	result := &Query{
		Includes: initIncludes(values),
		Fields:   initResourceFields(values),
		Sort:     initSort(values),
		Filters:  initFilters(values),
		Page:     initPage(values),
		Values:   values,
	}

	return result, nil
}

// ParseRequest parses the string into a path and a query,
// which are expected to be separated by a question mark '?'
// the path is parsed as follows:
// "/articles" -  article list request
// "/articles/42" - request of article with id 42
// "/articles/42/author" - request of an author related to the article with id 42
// "/article/42/relationships/author" - relationships request
//  see https://jsonapi.org/format/#document-resource-object-relationships
//
// for the query part description see "ParseQuery"
func ParseRequest(params string) (*Request, error) {
	path, query := split(params, '?', true)
	request, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	q, err := ParseQuery(query)
	if err != nil {
		return nil, err
	}
	request.Query = q
	return request, nil
}

func parsePath(path string) (*Request, error) {
	var err error
	path, err = url.PathUnescape(path)
	if err != nil {
		return nil, err
	}
	path = removeExtraDelimiters(path)
	if path == "" || (len(path) == 1 && path[0] == '/') {
		return nil, errors.New("qparser: empty path is given, path must have 1-4 segments")
	}
	if path[0] == '/' {
		path = path[1:]
	}
	requestParts := strings.Split(path, "/")
	request := new(Request)
	switch len(requestParts) {
	case 1:
		request.Resource.Type = requestParts[0]
	case 2:
		request.Resource.Type = requestParts[0]
		request.Resource.ID = requestParts[1]
	case 3:
		request.Resource.Type = requestParts[0]
		request.Resource.ID = requestParts[1]
		request.RelatedResourceType = requestParts[2]
	case 4:
		if requestParts[2] != relationshipsRequest {
			return nil, fmt.Errorf(
				"qparser: path format error, expected the segment 3 of the path is to be '%s' "+
					"but '%s' is received",
				relationshipsRequest,
				requestParts[2],
			)
		}
		request.Resource.Type = requestParts[0]
		request.Resource.ID = requestParts[1]
		request.RelationshipType = requestParts[3]
	default:
		return nil, fmt.Errorf("unknown path format %q, path must have 1-4 segments", path)
	}
	return request, nil
}

const (
	fieldsDelimiter = ","
	pageKeyword     = "page"
	sortKeyword     = "sort"
	filterKeyword   = "filter"
	includeKeyword  = "include"
	fieldsKeyword   = "fields"
)

func initResourceFields(values Values) ResourceFields {
	fieldsValues, ok := values[fieldsKeyword]
	if !ok {
		return nil
	}

	fields := make(ResourceFields)
	duplicates := make(map[string]map[string]struct{})
	returnFields := false
	for _, val := range fieldsValues {
		if val.Value == "" || len(val.NestedKeys) != 1 {
			continue
		}
		resourceType := val.NestedKeys[0]
		byResource, ok := duplicates[resourceType]
		if !ok {
			duplicates[resourceType] = make(map[string]struct{})
			byResource = duplicates[resourceType]
		}
		list := strings.Split(val.Value, fieldsDelimiter)
		toAppend := make([]string, 0, len(list))

		// append only not empty and unique values
		for _, item := range list {
			if item == "" {
				continue
			}
			if _, duplicated := byResource[item]; duplicated {
				continue
			}
			toAppend = append(toAppend, item)
			byResource[item] = struct{}{}
		}
		if len(toAppend) == 0 {
			continue
		}
		returnFields = true
		if _, ok := fields[resourceType]; ok {
			fields[resourceType] = append(fields[resourceType], toAppend...)
			continue
		}
		fields[resourceType] = toAppend

	}
	if returnFields {
		return fields
	}
	return nil
}

const (
	sortDelimiter = ','
	sortDescChar  = '-'
)

// initSort populates a list of sort fields and directions
// if a field name is prefixed by the '-' char then the sorting direction
// is treated as descending
func initSort(values Values) []Sort {
	sortValues, ok := values[sortKeyword]
	if !ok {
		return nil
	}
	sort := make([]Sort, 0)
	returnSort := false
	duplicates := make(map[string]struct{})
	for _, val := range sortValues {
		if val.Value == "" || len(val.NestedKeys) != 0 {
			continue
		}
		cur, rest := split(val.Value, sortDelimiter, true)
		for cur != "" {
			order := OrderAsc
			if cur[0] == sortDescChar {
				order = OrderDesc
				cur = cur[1:]
			}
			if _, exist := duplicates[cur]; exist {
				cur, rest = split(rest, sortDelimiter, true)
				continue
			}
			if cur == "" {
				cur, rest = split(rest, sortDelimiter, true)
				continue
			}
			returnSort = true
			duplicates[cur] = struct{}{}
			sort = append(
				sort,
				Sort{
					FieldName: cur,
					Order:     order,
				},
			)
			cur, rest = split(rest, sortDelimiter, true)
		}
	}
	if returnSort {
		return sort
	}
	return nil
}

// initFilters fills a list of filters
func initFilters(values Values) []Filter {
	filterValues, ok := values[filterKeyword]
	if !ok {
		return nil
	}
	filters := make([]Filter, 0)
	returnFilters := false

	for _, val := range filterValues {
		if val.Value == "" || len(val.NestedKeys) != 1 {
			continue
		}
		returnFilters = true
		filter := Filter{
			FieldName: val.NestedKeys[0],
			Predicate: val.Value,
		}
		filters = append(filters, filter)
	}
	if returnFilters {
		return filters
	}
	return nil
}

const (
	relationDelimiter       = ','
	nestedRelationDelimiter = '.'
)

// initIncludes creates a list of structures that can include nested lists
// with different depths these structures create a hierarchy on the basis of which
// the required inclusions can be implemented
// example:
//
//  query := "include=author,comments.author,comments.replies"
// 	values, err := ParseValues(query)
//	if err != nil {
//		return nil, err
//	}
//  includes := initIncludes(values)
//  ...
// [
//  {
//    "Relation": "author",
//    "Includes": null
//  },
//  {
//    "Relation": "comments",
//    "Includes": [
//      {
//        "Relation": "author",
//        "Includes": null
//      },
//      {
//        "Relation": "replies",
//        "Includes": null
//      }
//    ]
//  }
//]
func initIncludes(values Values) []Include {
	incValues, ok := values[includeKeyword]
	if !ok {
		return nil
	}
	// comments,comments.author.image,comments.author.posts
	roots := make(map[string]*Include)
	// this slice is needed in order to preserve order of includes
	ordered := make([]*Include, 0)
	for _, val := range incValues {
		if len(val.NestedKeys) > 0 || val.Value == "" {
			continue
		}
		cur, rest := split(val.Value, relationDelimiter, true)
		for cur != "" {
			var root *Include
			rootKey, next := split(cur, nestedRelationDelimiter, true)
			if existingRoot, ok := roots[rootKey]; ok {
				root = existingRoot
			} else {
				root = &Include{Relation: rootKey}
				roots[rootKey] = root
				ordered = append(ordered, root)
			}
			expandInclude(root, next)
			cur, rest = split(rest, relationDelimiter, true)
		}
	}
	includes := make([]Include, 0, len(ordered))
	for _, include := range ordered {
		includes = append(includes, *include)
	}
	return includes
}

func expandInclude(root *Include, queryPart string) {
	if queryPart == "" {
		return
	}
	var newRoot *Include // to pass down in recursion
	cur, rest := split(queryPart, nestedRelationDelimiter, true)
	// if no include then create a new slice
	if root.Includes == nil {
		root.Includes = []Include{{Relation: cur}}
		newRoot = &root.Includes[0]
	} else {
		// check if the include with the same relation already exists and use it if so
		for i := 0; i < len(root.Includes); i++ {
			if root.Includes[i].Relation == cur {
				newRoot = &root.Includes[i]
				break
			}
		}
		// otherwise add a new one
		if newRoot == nil {
			root.Includes = append(root.Includes, Include{Relation: cur})
			newRoot = &root.Includes[len(root.Includes)-1]
		}
	}
	expandInclude(newRoot, rest)
}

func initPage(values Values) *Page {
	pageValues, ok := values[pageKeyword]
	if !ok {
		return nil
	}
	returnPage := false
	page := new(Page)
	for _, val := range pageValues {
		if len(val.NestedKeys) != 1 {
			continue
		}
		switch val.NestedKeys[0] {
		case "size":
			returnPage = true
			page.Size = val.Value
		case "number":
			returnPage = true
			page.Number = val.Value
		case "limit":
			returnPage = true
			page.Limit = val.Value
		case "offset":
			returnPage = true
			page.Offset = val.Value
		case "cursor":
			returnPage = true
			page.Cursor = val.Value
		}
	}
	if returnPage {
		return page
	}
	return nil
}

const (
	openBracket     = '['
	closeBracket    = ']'
	nestedKeyDefMin = 3 // 3 characters is minimal length for nested key definition e.g. "[k]"
)

// extractKeys fetches top and nested keys from the passed string
// for example string "top[n1][n2]" will result in return values: "top", []string{"n1", "n2"}
// if there is no nested keys then the second return value would be nil
// nested keys must be enclosed in square brackets, double opening or closing square brackets or any characters
// between the closing and opening brackets are not allowed
// any violation of this syntax is interpreted as absence of nested keys and the
// given argument string is returned as a top-level key unchanged
func extractKeys(key string) (string, []string) {
	if key == "" {
		return key, nil
	}
	var rest string
	topKey := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c == closeBracket || i == 0 && c == openBracket {
			return key, nil
		}
		if c == openBracket {
			rest = key[i:]
			break
		}
		topKey = append(topKey, c)
	}
	if rest == "" || len(rest) < nestedKeyDefMin {
		return key, nil
	}
	nestedKey := make([]byte, 0, 16)
	nestedKeys := make([]string, 0, 4)
	opened := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if opened && c == openBracket || !opened && c != openBracket {
			return key, nil
		}
		switch c {
		case openBracket:
			opened = true
			continue
		case closeBracket:
			if len(nestedKey) == 0 {
				return key, nil
			}
			opened = false
			nestedKeys = append(nestedKeys, string(nestedKey))
			nestedKey = nestedKey[:0]
			continue
		}
		nestedKey = append(nestedKey, c)
	}
	return string(topKey), nestedKeys
}

// split slices s into two substrings separated by the first occurrence of
// sep. If cutc is true then sep is excluded from the second substring.
// If sep does not occur in s then s and the empty string is returned.
func split(s string, sep byte, cutc bool) (string, string) {
	i := strings.IndexByte(s, sep)
	if i < 0 {
		return s, ""
	}
	if cutc {
		return s[:i], s[i+1:]
	}
	return s[:i], s[i:]
}

// removeExtraDelimiters clears the string from the following delimiter characters
func removeExtraDelimiters(path string) string {
	const delim = '/'

	if path == "" {
		return path
	}
	rm := path[0] == delim
	result := []byte{path[0]}
	for i := 1; i < len(path); i++ {
		c := path[i]
		if c == delim && rm {
			continue
		}
		result = append(result, c)
		rm = c == delim
	}
	return string(result)
}
