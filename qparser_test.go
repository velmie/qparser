package qparser

import (
	"reflect"
	"strings"
	"testing"
)

type pathTest struct {
	in          string
	out         *Request
	errContains string
}

var pathTests = []pathTest{
	{
		in: "/articles",
		out: &Request{
			Resource: Resource{Type: "articles"},
		},
	},
	{
		in: "/articles/1",
		out: &Request{
			Resource: Resource{Type: "articles", ID: "1"},
		},
	},
	{
		in: "/articles/bd98b83f-dd5b-4cab-884b-91bf0faadf7a",
		out: &Request{
			Resource: Resource{Type: "articles", ID: "bd98b83f-dd5b-4cab-884b-91bf0faadf7a"},
		},
	},
	{
		in: "/articles/1/author",
		out: &Request{
			Resource:            Resource{Type: "articles", ID: "1"},
			RelatedResourceType: "author",
		},
	},
	{
		in: "/articles/1/relationships/comments",
		out: &Request{
			Resource:         Resource{Type: "articles", ID: "1"},
			RelationshipType: "comments",
		},
	},
	{
		in:          "/",
		errContains: "empty path",
	},
	{
		in:          "/",
		errContains: "empty path",
	},
	{
		in:          "/resource/id/should_be_relationships/relationship_type",
		errContains: relationshipsRequest,
	},
	{
		in:          "///",
		errContains: "empty path",
	},
	{
		in: "///articles//1//",
		out: &Request{
			Resource: Resource{Type: "articles", ID: "1"},
		},
	},
}

func TestPathParsing(t *testing.T) {
	for _, tt := range pathTests {
		r, err := parsePath(tt.in)

		if err != nil && tt.errContains == "" {
			t.Errorf("parsePath(%q) returned unexpected error %s", tt.in, err)
			continue
		}
		if err == nil && tt.errContains != "" {
			t.Errorf(
				"expected parsePath(%q) to return error which contains %q, but nil is returned",
				tt.in,
				tt.errContains,
			)
			continue
		}
		if !reflect.DeepEqual(r, tt.out) {
			t.Errorf("parsePath(%q):\n\tgot  %+v\n\twant %+v\n", tt.in, r, tt.out)
		}
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if tt.errContains != "" {
			if !strings.Contains(errStr, tt.errContains) {
				t.Errorf(
					`parsePath(%q) returned error %q, want something containing %q"`,
					tt.in,
					errStr,
					tt.errContains)
			}
		}
	}
}

type removeDelimiterTest struct {
	in  string
	out string
}

var removeDelimiterTests = []removeDelimiterTest{
	{
		"///",
		"/",
	},
	{
		"",
		"",
	},
	{
		"/a/b/c",
		"/a/b/c",
	},
	{
		"/a/b/c/",
		"/a/b/c/",
	},
	{
		"/a/b/c//",
		"/a/b/c/",
	},
	{
		"////a////b/////c//こんにちは",
		"/a/b/c/こんにちは",
	},
}

func TestRemoveExtraDelimiters(t *testing.T) {
	for _, tt := range removeDelimiterTests {
		if r := removeExtraDelimiters(tt.in); r != tt.out {
			t.Errorf(
				`removeExtraDelimiters(%q) returned %q, want %q"`,
				tt.in,
				r,
				tt.out,
			)
		}
	}
}

type parseValuesTest struct {
	in  string
	out Values
}

var parseValuesTests = []parseValuesTest{
	{
		in:  "",
		out: Values{},
	},
	{
		in: "key=value&key=value2",
		out: Values{
			"key": {
				{
					TopLevelKey: "key",
					Value:       "value",
				},
				{
					TopLevelKey: "key",
					Value:       "value2",
				},
			},
		},
	},
	{
		in: "key=value&key[property]=param;key[property][sub]=param2",
		out: Values{
			"key": {
				{
					TopLevelKey: "key",
					Value:       "value",
				},
				{
					TopLevelKey: "key",
					NestedKeys:  []string{"property"},
					Value:       "param",
				},
				{
					TopLevelKey: "key",
					NestedKeys:  []string{"property", "sub"},
					Value:       "param2",
				},
			},
		},
	},
	{
		in: "key[property]=value&&&&&&;;;;;key[property]=value2",
		out: Values{
			"key": {
				{
					TopLevelKey: "key",
					NestedKeys:  []string{"property"},
					Value:       "value",
				},
				{
					TopLevelKey: "key",
					NestedKeys:  []string{"property"},
					Value:       "value2",
				},
			},
		},
	},
	{
		in: "?some&separated;entries",
		out: Values{
			"some": {
				Value{
					TopLevelKey: "some",
				},
			},
			"separated": {
				Value{
					TopLevelKey: "separated",
				},
			},
			"entries": {
				Value{
					TopLevelKey: "entries",
				},
			},
		},
	},
}

func TestParseValues(t *testing.T) {
	for _, tt := range parseValuesTests {
		values, err := ParseValues(tt.in)
		if err != nil {
			t.Errorf("ParseValues(%q) returned error %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(values, tt.out) {
			t.Errorf(
				"ParseValues(%q):\n\tgot  %+v\n\twant %+v\n",
				tt.in,
				values,
				tt.out,
			)
		}
	}
}

type extractKeysTest = struct {
	in            string
	outTopKey     string
	outNestedKeys []string
}

var extractKeysTests = []extractKeysTest{
	{
		in:        "field",
		outTopKey: "field",
	},
	{
		in:            "field[lvl1][lvl2][lvl3]",
		outTopKey:     "field",
		outNestedKeys: []string{"lvl1", "lvl2", "lvl3"},
	},
	{
		in:            "field[lvl1]",
		outTopKey:     "field",
		outNestedKeys: []string{"lvl1"},
	},
	{
		in:            "field[ ]",
		outTopKey:     "field",
		outNestedKeys: []string{" "},
	},
	{
		in:            " [ ]",
		outTopKey:     " ",
		outNestedKeys: []string{" "},
	},
	{
		in:        " [ ].",
		outTopKey: " [ ].",
	},
	{
		in:        "field[lvl1][lvl2].[lvl3]",
		outTopKey: "field[lvl1][lvl2].[lvl3]",
	},
	{
		in:            "field    [lvl1][lvl2][lvl3]",
		outTopKey:     "field    ",
		outNestedKeys: []string{"lvl1", "lvl2", "lvl3"},
	},
	{
		in:            "k[n]",
		outTopKey:     "k",
		outNestedKeys: []string{"n"},
	},
	{
		in:            "field[ッ][!@#$%^&*().]",
		outTopKey:     "field",
		outNestedKeys: []string{"ッ", "!@#$%^&*()."},
	},
	{
		in:        "k[]",
		outTopKey: "k[]",
	},
}

func TestExtractKeys(t *testing.T) {
	for _, tt := range extractKeysTests {
		top, nested := extractKeys(tt.in)

		if tt.outTopKey != top {
			t.Errorf("extractKeys(%q) returned top key %q, expected top key to be %q", tt.in, top, tt.outTopKey)
		}
		if nested == nil && tt.outNestedKeys != nil {
			t.Errorf(
				"extractKeys(%q) returned 'nil' nested keys slice, expected nested keys slice to have values %+v",
				tt.in,
				tt.outNestedKeys,
			)
			continue
		}
		if nested != nil && tt.outNestedKeys == nil {
			t.Errorf(
				"extractKeys(%q) returned not nil nested keys slice %+v, expected nested keys slice to be nil",
				tt.in,
				nested,
			)
			continue
		}

		if !reflect.DeepEqual(nested, tt.outNestedKeys) {
			t.Errorf(
				"extractKeys(%q):\n\tgot nested keys  %+v\n\twant nested keys %+v\n",
				tt.in,
				nested,
				tt.outNestedKeys,
			)
		}
	}
}

var extractKeysBenchmarks = []string{
	"topkeyonly",
	"key[one_nested]",
	"key[nested1][nested2]",
	"key[and][syntax]violation",
}

func BenchmarkExtractKeys(b *testing.B) {
	for _, arg := range extractKeysBenchmarks {
		b.Run(arg, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				extractKeys(arg)
			}
			b.StopTimer()
		})
	}
}

type valuesGetTest struct {
	in     string
	nested []string
	out    string
}

var (
	values = Values{
		"page": []Value{
			{
				TopLevelKey: "page",
				NestedKeys:  []string{"size"},
				Value:       "10",
			},
			{
				TopLevelKey: "page",
				NestedKeys:  []string{"number"},
				Value:       "2",
			},
			{
				TopLevelKey: "page",
				NestedKeys:  []string{"header", "title"},
				Value:       "Title",
			},
			{
				TopLevelKey: "page",
				NestedKeys:  []string{"header", "font", "name"},
				Value:       "Helvetica",
			},
		},
		"include": []Value{
			{
				TopLevelKey: "include",
				Value:       "author,comments",
			},
		},
	}
	valuesGetTests = []valuesGetTest{
		{
			in:  "page",
			out: "",
		},
		{
			in:     "page",
			nested: []string{"size"},
			out:    "10",
		},
		{
			in:     "page",
			nested: []string{"number"},
			out:    "2",
		},
		{
			in:     "page",
			nested: []string{"header"},
			out:    "",
		},
		{
			in:     "page",
			nested: []string{"header", "title"},
			out:    "Title",
		},
		{
			in:     "page",
			nested: []string{"header", "font"},
			out:    "",
		},
		{
			in:     "page",
			nested: []string{"header", "font", "name"},
			out:    "Helvetica",
		},
		{
			in:  "unknown",
			out: "",
		},
		{
			in:  "include",
			out: "author,comments",
		},
		{
			in:     "include",
			nested: make([]string, 0),
			out:    "author,comments",
		},
		{
			in:     "include",
			nested: make([]string, 1),
			out:    "",
		},
	}
)

func TestValuesGet(t *testing.T) {
	for _, tt := range valuesGetTests {
		if r := values.Get(tt.in, tt.nested...); r != tt.out {
			t.Errorf(
				`values.Get(%q) returned %q, want %q"`,
				tt.in,
				r,
				tt.out,
			)
		}
	}
}

type initPageTest struct {
	in  Values
	out *Page
}

var initPageTests = []initPageTest{
	{
		in:  Values{},
		out: nil,
	},
	{
		in: Values{
			"not_a_page_key": []Value{
				{
					TopLevelKey: "not_a_page_key",
					Value:       "1",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"page": []Value{
				{
					TopLevelKey: "page",
					Value:       "1",
					NestedKeys:  []string{"unknown"},
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"page": []Value{
				{
					TopLevelKey: "page",
					Value:       "1",
					NestedKeys:  []string{"size", "more"},
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"page": []Value{
				{
					TopLevelKey: "page",
					Value:       "1",
					NestedKeys:  []string{"size"},
				},
			},
		},
		out: &Page{Size: "1"},
	},
	{
		in: Values{
			"page": []Value{
				{
					TopLevelKey: "page",
					Value:       "10",
					NestedKeys:  []string{"size"},
				},
				{
					TopLevelKey: "page",
					Value:       "5",
					NestedKeys:  []string{"number"},
				},
				{
					TopLevelKey: "page",
					Value:       "30",
					NestedKeys:  []string{"limit"},
				},
				{
					TopLevelKey: "page",
					Value:       "90",
					NestedKeys:  []string{"offset"},
				},
				{
					TopLevelKey: "page",
					Value:       "436961cb-2ed1-4d53-a554-a3ee9ed55223",
					NestedKeys:  []string{"cursor"},
				},
				{
					TopLevelKey: "page",
					Value:       "undefined",
					NestedKeys:  []string{"value"},
				},
			},
		},
		out: &Page{
			Size:   "10",
			Number: "5",
			Limit:  "30",
			Offset: "90",
			Cursor: "436961cb-2ed1-4d53-a554-a3ee9ed55223",
		},
	},
}

func TestInitPage(t *testing.T) {
	for _, tt := range initPageTests {
		page := initPage(tt.in)
		if !reflect.DeepEqual(page, tt.out) {
			t.Errorf(
				"initPage(%+v):\n\tgot  %+v\n\twant %+v\n",
				tt.in,
				page,
				tt.out,
			)
		}
	}
}

type initIncludesTest struct {
	in  Values
	out []Include
}

var initIncludesTests = []initIncludesTest{
	{
		in:  Values{},
		out: nil,
	},
	{
		in: Values{"not_an_include_key": {
			Value{
				TopLevelKey: "not_an_include_key",
				Value:       "val",
			},
		}},
		out: nil,
	},
	{
		in: Values{"include": {
			Value{
				TopLevelKey: "include",
				Value:       "author",
			},
		}},
		out: []Include{
			{
				Relation: "author",
				Includes: nil,
			},
		},
	},
	{
		in: Values{"include": {
			Value{
				TopLevelKey: "include",
				Value:       "author,logo",
			},
		}},
		out: []Include{
			{
				Relation: "author",
				Includes: nil,
			},
			{
				Relation: "logo",
				Includes: nil,
			},
		},
	},
	{
		in: Values{"include": {
			Value{
				TopLevelKey: "include",
				Value:       "author,comments.author,comments.replies",
			},
		}},
		out: []Include{
			{
				Relation: "author",
				Includes: nil,
			},
			{
				Relation: "comments",
				Includes: []Include{
					{
						Relation: "author",
						Includes: nil,
					},
					{
						Relation: "replies",
						Includes: nil,
					},
				},
			},
		},
	},
	{
		in: Values{"include": {
			Value{
				TopLevelKey: "include",
				Value:       "author,comments.author",
			},
			Value{
				TopLevelKey: "include",
				Value:       "comments.replies",
			},
			Value{
				TopLevelKey: "include",
				Value:       "author.avatar",
			},
		}},
		out: []Include{
			{
				Relation: "author",
				Includes: []Include{
					{
						Relation: "avatar",
						Includes: nil,
					},
				},
			},
			{
				Relation: "comments",
				Includes: []Include{
					{
						Relation: "author",
						Includes: nil,
					},
					{
						Relation: "replies",
						Includes: nil,
					},
				},
			},
		},
	},
	{
		in: Values{"include": {
			Value{
				TopLevelKey: "include",
				Value:       "duplicate,duplicate,duplicate",
			},
		}},
		out: []Include{
			{
				Relation: "duplicate",
				Includes: nil,
			},
		},
	},
	{
		in: Values{"include": {
			Value{
				TopLevelKey: "include",
				Value:       "duplicate",
			},
			Value{
				TopLevelKey: "include",
				Value:       "duplicate,duplicate,duplicate",
			},
		}},
		out: []Include{
			{
				Relation: "duplicate",
				Includes: nil,
			},
		},
	},
}

func TestInitIncludes(t *testing.T) {
	for _, tt := range initIncludesTests {
		includes := initIncludes(tt.in)
		if !reflect.DeepEqual(includes, tt.out) {
			t.Errorf(
				"initIncludes(%+v):\n\tgot  %+v\n\twant %+v\n",
				tt.in,
				includes,
				tt.out,
			)
		}
	}
}

type initSortTest struct {
	in  Values
	out []Sort
}

var initSortTests = []initSortTest{
	{
		in: Values{
			"not_a_sort_key": {
				Value{
					TopLevelKey: "not_a_sort_key",
					Value:       "val",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "val",
					NestedKeys:  []string{"nested"},
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       ",,,",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "createdAt",
				},
			},
		},
		out: []Sort{
			{
				FieldName: "createdAt",
				Order:     OrderAsc,
			},
		},
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "-createdAt",
				},
			},
		},
		out: []Sort{
			{
				FieldName: "createdAt",
				Order:     OrderDesc,
			},
		},
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "createdAt,-title",
				},
			},
		},
		out: []Sort{
			{
				FieldName: "createdAt",
				Order:     OrderAsc,
			},
			{
				FieldName: "title",
				Order:     OrderDesc,
			},
		},
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "createdAt,-title",
				},
				Value{
					TopLevelKey: "sort",
					Value:       "author",
				},
			},
		},
		out: []Sort{
			{
				FieldName: "createdAt",
				Order:     OrderAsc,
			},
			{
				FieldName: "title",
				Order:     OrderDesc,
			},
			{
				FieldName: "author",
				Order:     OrderAsc,
			},
		},
	},
	{
		in: Values{
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "createdAt,-title,duplicate,duplicate",
				},
				Value{
					TopLevelKey: "sort",
					Value:       "author,-duplicate",
				},
			},
		},
		out: []Sort{
			{
				FieldName: "createdAt",
				Order:     OrderAsc,
			},
			{
				FieldName: "title",
				Order:     OrderDesc,
			},
			{
				FieldName: "duplicate",
				Order:     OrderAsc,
			},
			{
				FieldName: "author",
				Order:     OrderAsc,
			},
		},
	},
}

func TestInitSort(t *testing.T) {
	for _, tt := range initSortTests {
		sorts := initSort(tt.in)
		if !reflect.DeepEqual(sorts, tt.out) {
			t.Errorf(
				"initSort(%+v):\n\tgot  %+v\n\twant %+v\n",
				tt.in,
				sorts,
				tt.out,
			)
		}
	}
}

type initFiltersTest struct {
	in  Values
	out []Filter
}

var initFiltersTests = []initFiltersTest{
	{
		in:  Values{},
		out: nil,
	},
	{
		in: Values{
			"not_a_filter": []Value{
				{
					TopLevelKey: "not_a_filter",
					Value:       "val",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"filter": []Value{
				{
					TopLevelKey: "filter",
					NestedKeys:  nil, //! no nested keys
					Value:       "val",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"filter": []Value{
				{
					TopLevelKey: "filter",
					NestedKeys:  []string{"too", "many"},
					Value:       "val",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"filter": []Value{
				{
					TopLevelKey: "filter",
					NestedKeys:  []string{"createdAt"},
					Value:       "lt:2020-01-02",
				},
				{
					TopLevelKey: "filter",
					NestedKeys:  []string{"title"},
					Value:       "like:poker",
				},
			},
		},
		out: []Filter{
			{
				FieldName: "createdAt",
				Predicate: "lt:2020-01-02",
			},
			{
				FieldName: "title",
				Predicate: "like:poker",
			},
		},
	},
	{
		in: Values{
			"filter": []Value{
				{
					TopLevelKey: "filter",
					NestedKeys:  []string{"title"},
					Value:       "eq:foo",
				},
				{
					TopLevelKey: "filter",
					NestedKeys:  []string{"title"},
					Value:       "eq:bar",
				},
			},
		},
		out: []Filter{
			{
				FieldName: "title",
				Predicate: "eq:foo",
			},
			{
				FieldName: "title",
				Predicate: "eq:bar",
			},
		},
	},
}

func TestInitFilters(t *testing.T) {
	for _, tt := range initFiltersTests {
		filter := initFilters(tt.in)
		if !reflect.DeepEqual(filter, tt.out) {
			t.Errorf(
				"initFilters(%+v):\n\tgot  %+v\n\twant %+v\n",
				tt.in,
				filter,
				tt.out,
			)
		}
	}
}

type initResourceFieldsTest struct {
	in  Values
	out ResourceFields
}

var initResourceFieldsTests = []initResourceFieldsTest{
	{
		in:  Values{},
		out: nil,
	},
	{
		in: Values{
			"not_a_fields_key": []Value{
				{
					TopLevelKey: "not_a_fields_key",
					Value:       "val",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "no_nested_key",
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "val",
					NestedKeys:  []string{"more", "than", "one", "nested", "key"},
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       ",,,,,",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: nil,
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       ",,,title,,",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: ResourceFields{
			"articles": []string{"title"},
		},
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "title",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: ResourceFields{
			"articles": []string{"title"},
		},
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "title,body,image",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: ResourceFields{
			"articles": []string{"title", "body", "image"},
		},
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "title,body,image,duplicate,duplicate",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: ResourceFields{
			"articles": []string{"title", "body", "image", "duplicate"},
		},
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "title,body,image",
					NestedKeys:  []string{"articles"},
				},
				{
					TopLevelKey: "fields",
					Value:       "title,body,topic,image",
					NestedKeys:  []string{"articles"},
				},
			},
		},
		out: ResourceFields{
			"articles": []string{"title", "body", "image", "topic"},
		},
	},
	{
		in: Values{
			"fields": []Value{
				{
					TopLevelKey: "fields",
					Value:       "title,body,image",
					NestedKeys:  []string{"articles"},
				},
				{
					TopLevelKey: "fields",
					Value:       "title,id",
					NestedKeys:  []string{"comments"},
				},
			},
		},
		out: ResourceFields{
			"articles": []string{"title", "body", "image"},
			"comments": []string{"title", "id"},
		},
	},
}

func TestInitResourceFields(t *testing.T) {
	for _, tt := range initResourceFieldsTests {
		fields := initResourceFields(tt.in)
		if !reflect.DeepEqual(fields, tt.out) {
			t.Errorf(
				"initResourceFields(%+v):\n\tgot  %+v\n\twant %+v\n",
				tt.in,
				fields,
				tt.out,
			)
		}
	}
}

func TestParseQuery(t *testing.T) {
	const query = "?filter[title]=eq:foo&page[size]=16&sort=-createdAt,title&include=author&fields[articles]=title,body"
	expected := &Query{
		Includes: []Include{
			{
				Relation: "author",
			},
		},
		Fields: ResourceFields{
			"articles": []string{"title", "body"},
		},
		Sort: []Sort{
			{
				FieldName: "createdAt",
				Order:     OrderDesc,
			},
			{
				FieldName: "title",
				Order:     OrderAsc,
			},
		},
		Filters: []Filter{
			{
				FieldName: "title",
				Predicate: "eq:foo",
			},
		},
		Page: &Page{
			Size: "16",
		},
		Values: Values{
			"filter": {
				Value{
					TopLevelKey: "filter",
					NestedKeys:  []string{"title"},
					Value:       "eq:foo",
				},
			},
			"page": {
				Value{
					TopLevelKey: "page",
					NestedKeys:  []string{"size"},
					Value:       "16",
				},
			},
			"sort": {
				Value{
					TopLevelKey: "sort",
					Value:       "-createdAt,title",
				},
			},
			"include": {
				Value{
					TopLevelKey: "include",
					Value:       "author",
				},
			},
			"fields": {
				Value{
					TopLevelKey: "fields",
					NestedKeys:  []string{"articles"},
					Value:       "title,body",
				},
			},
		},
	}

	got, err := ParseQuery(query)
	if err != nil {
		t.Errorf("ParseQuery(%q) returned error %v", query, err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf(
			"ParseQuery(%q):\n\tgot  %+v\n\twant %+v\n",
			query,
			got,
			expected,
		)
	}
}

func TestParseRequest(t *testing.T) {
	const request = "/articles/42/comments?fields[comments]=author"
	expected := &Request{
		Resource: Resource{
			Type: "articles",
			ID:   "42",
		},
		RelatedResourceType: "comments",
		Query: &Query{
			Fields: ResourceFields{
				"comments": []string{"author"},
			},
			Values: Values{
				"fields": {
					Value{
						TopLevelKey: "fields",
						NestedKeys:  []string{"comments"},
						Value:       "author",
					},
				},
			},
		},
	}

	got, err := ParseRequest(request)
	if err != nil {
		t.Errorf("ParseRequest(%q) returned error %v", request, err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf(
			"ParseRequest(%q):\n\tgot  %+v\n\twant %+v\n",
			request,
			got,
			expected,
		)
	}
}
