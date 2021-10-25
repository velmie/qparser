# QParser

The package helps to parse part of the URL path and its parameters string to a handy structure.
The structure format is compatible with the [JSON:API](https://jsonapi.org/) specification. 
QParser will be useful both for implementing the API according to this specification and independently.

```go
	params := "/articles/42?fields[articles]=title,body&include=comments.author&filter[createdAt]=lt:2015-10-02&sort=-createdAt"

	request, _ := qparser.ParseRequest(params)

	values := request.Query.Values
	fmt.Printf("1. raw value of fields[articles] is %q\n", values.Get("fields", "articles"))
	fmt.Printf("2. the list of requested fields for the articles resource is %v\n", request.Query.Fields["articles"])

	sort := request.Query.Sort
	if len(sort) > 0 {
		fmt.Printf("3. sort by %s %s\n", sort[0].FieldName, sort[0].Order)
		fmt.Printf("4. is descending order? %t", sort[0].Order == qparser.OrderDesc)
	}

	payload, _ := json.MarshalIndent(request, "", "  ")
	fmt.Printf("\nThe structure:\n\n%s", payload)
```

The example will print:

```text
1. raw value of fields[articles] is "title,body"
2. the list of requested fields for the articles resource is [title body]
3. sort by createdAt DESC
4. is descending order? true
The structure:

{
  "Resource": {
    "Type": "articles",
    "ID": "42"
  },
  "RelationshipType": "",
  "RelatedResourceType": "",
  "Query": {
    "Includes": [
      {
        "Relation": "comments",
        "Includes": [
          {
            "Relation": "author",
            "Includes": null
          }
        ]
      }
    ],
    "Fields": {
      "articles": [
        "title",
        "body"
      ]
    },
    "Sort": [
      {
        "FieldName": "createdAt",
        "Order": 1
      }
    ],
    "Page": null,
    "Values": {
      "fields": [
        {
          "TopLevelKey": "fields",
          "NestedKeys": [
            "articles"
          ],
          "Value": "title,body"
        }
      ],
      "filter": [
        {
          "TopLevelKey": "filter",
          "NestedKeys": [
            "createdAt"
          ],
          "Value": "lt:2015-10-02"
        }
      ],
      "include": [
        {
          "TopLevelKey": "include",
          "NestedKeys": null,
          "Value": "comments.author"
        }
      ],
      "sort": [
        {
          "TopLevelKey": "sort",
          "NestedKeys": null,
          "Value": "-createdAt"
        }
      ]
    }
  }
}
```

## Using the "Values"

The package fundamental types are the "Value" struct, and the "Values" map.
The Value type is used to represent key-value pairs "key=value",  
with the addition of "nested keys"  "key\[nested_key\]=value". 
The types provide convenient way for accessing properties of the query string of this kind "style\[top\]\[color\]=white&style\[size\]=XL".

The string is split into substrings separated by ampersands '&' or semicolons ';'. 
The key is the part of the substring up to the equal sign '='. 
Anything that goes after the equal sign is interpreted as a value.
Substrings in the key part surrounded by square brackets '\ [', '\]' are interpreted as nested keys.

A setting without an equals sign is interpreted as a key set to an empty value.

To get a map of values from a string, call the "*ParseValues*" function.

```go
	q := "style[top][color]=white&style[size]=XL"
	values, _ := qparser.ParseValues(q)

	fmt.Printf(
		"Top color is %q, size is %q\n",
		values.Get("style", "top", "color"),
		values.Get("style", "size"),
	)
    // prints: Top color is "white", size is "XL"
```

To access multiple values or check a key presence use the map directly. 
```go
	q := "size=L&size=XL"
	values, _ := qparser.ParseValues(q)

	if list, ok := values["size"]; ok {
		fmt.Println("the list of sizes: ")
		for _, size := range list {
			if len(size.NestedKeys) > 0 {
				continue
			}
			fmt.Println(size.Value)
		}
	}
    // prints:
    //
    // the list of sizes: 
    // L
    // XL
```

## The "Query" structure

The "Query" structure adds some extras. The "*ParseQuery*" function additionally processes 
the values of the following query keys:

* include
* fields\[resource_type\]
* sort
* filter\[field_name\]
* page

### Includes

The value of the *include* key is considered as a request to add resources related to the requested resource.
For example, a client requests an article and asks to include in the response the data of the author of the article, 
comments on this article and data about the author of the comment.
The request might look like this "articles/42?include=author,comments.author".

```go
	pathAndQuery := "articles/42?include=author,comments.author"
	q := pathAndQuery[strings.IndexByte(pathAndQuery, '?'):] // separate the path from the query

	query, _ := qparser.ParseQuery(q)

	includes, _ := json.MarshalIndent(query.Includes, "", "  ")
	fmt.Println(string(includes))
```

The above example outputs:
```json
[
  {
    "Relation": "author",
    "Includes": null
  },
  {
    "Relation": "comments",
    "Includes": [
      {
        "Relation": "author",
        "Includes": null
      }
    ]
  }
]
```
The hierarchy of this recursive structure represents the resources that needs to be included in the response.

The calling code can iterate over this structure to implement the desired data loads. 
> Note that QParser does not limit the depth of inclusions. 
Any constraints and checks must be done in the calling code.


### Fields

It is assumed that the fields query parameter is used to specify the list of attributes of the requested resource.
The "fields" parameter must specify the resource type as a nested key e.g. "fields\[articles\]=title,body".
There should be exactly one nested key. Any other form of the "fields" parameter is ignored.

```go
	q := "fields[articles]=title,body&fields[author]=name,dob"

	query, _ := qparser.ParseQuery(q)

	fields, _ := json.MarshalIndent(query.Fields, "", "  ")
	fmt.Println(string(fields))
```
Prints:

```json
{
  "articles": [
    "title",
    "body"
  ],
  "author": [
    "name",
    "dob"
  ]
}
```

### Sort

The value of the "sort" query parameter represents sort fields separated by the comma.
The sort order is ascending by default. In order to specify descending sort order the sort field must be
prefixed with the minus sign. 
For instance "sort=-createdAt,title" means to sort a list from the latest to newest and then by title in the ascending order.

```go
	q := "sort=-createdAt,title"

	query, _ := qparser.ParseQuery(q)

	for _, sort := range query.Sort {
		fmt.Printf("sort by %q %q \n", sort.FieldName, sort.Order)
		fmt.Printf(
			"Ascending: %t, Descending: %t\n",
			sort.Order == qparser.OrderAsc,
			sort.Order == qparser.OrderDesc,
		)
	}
```

Prints:

```text
sort by "createdAt" "DESC" 
Ascending: false, Descending: true
sort by "title" "ASC" 
Ascending: true, Descending: false
```

### Filters

For convenience QParser fills the filter list if the "filter" keyword is present in the query string with exactly 1 
nested key which is interpreted as a field name. The value of this parameter is considered a predicate.
The interpretation of the predicate must be implemented by the calling code.
For example "filter\[company\]=eq:Velmie". 
If your filter implementation intends to use nested keys, like so "filter\[company\]\[eq\]=Velmie",
then use the "Values" directly.

```go
	q := "filter[company]=eq:Velmie&filter[date]=notnull"

	query, _ := qparser.ParseQuery(q)

	filters, _ := json.MarshalIndent(query.Filters, "", "  ")
	fmt.Println(string(filters))
```

Prints:

```json
[
  {
    "FieldName": "company",
    "Predicate": "eq:Velmie"
  },
  {
    "FieldName": "date",
    "Predicate": "notnull"
  }
]
```

### Page

It is assumed that the page parameter will be used to implement pagination. 
QParser does not enforce a specific pagination implementation, therefore the "Page" structure contains the most popular terms: limit, offset; number, size; cursor.
QParser fills the given structure with the corresponding values from page\[limit\], page\[offset\] etc.

```go
	q := "page[size]=32&page[number]=8"

	query, _ := qparser.ParseQuery(q)

	page, _ := json.MarshalIndent(query.Page, "", "  ")
	fmt.Println(string(page))
```

Prints:

```json
{
  "Size": "32",
  "Number": "8",
  "Limit": "",
  "Offset": "",
  "Cursor": ""
}
```

## The "Request" structure

The Request structure can be useful when implementing API endpoints URLs following recommendations
from the JSON:API specification. 
See the page, [https://jsonapi.org/recommendations/#urls](https://jsonapi.org/recommendations/#urls).
