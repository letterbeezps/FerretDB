// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/FerretDB/FerretDB/integration/setup"
)

func TestAggregateAddFieldsErrors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct { //nolint:vet // used for test only
		pipeline bson.A // required, aggregation pipeline stages

		err        *mongo.CommandError // required
		altMessage string              // optional, alternative error message
		skip       string              // optional, skip test with a specified reason
	}{
		"NotDocument": {
			pipeline: bson.A{
				bson.D{{"$addFields", "not-document"}},
			},
			err: &mongo.CommandError{
				Code:    40272,
				Name:    "Location40272",
				Message: "$addFields specification stage must be an object, got string",
			},
			altMessage: "$addFields specification stage must be an object, got string",
		},
		"InvalidFieldPath": {
			pipeline: bson.A{
				bson.D{{"$addFields", bson.D{{"$foo", "v"}}}},
			},
			err: &mongo.CommandError{
				Code:    16410,
				Name:    "Location16410",
				Message: "Invalid $addFields :: caused by :: FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			require.NotNil(t, tc.pipeline, "pipeline must not be nil")
			require.NotNil(t, tc.err, "err must not be nil")

			ctx, collection := setup.Setup(t)

			_, err := collection.Aggregate(ctx, tc.pipeline)

			if tc.altMessage != "" {
				AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
				return
			}

			AssertEqualCommandError(t, *tc.err, err)
		})
	}
}

func TestAggregateGroupErrors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		pipeline bson.A // required, aggregation pipeline stages

		err        *mongo.CommandError // required, expected error from MongoDB
		altMessage string              // optional, alternative error message for FerretDB, ignored if empty
		skip       string              // optional, skip test with a specified reason
	}{
		"StageGroupUnaryOperatorSum": {
			pipeline: bson.A{
				bson.D{{"$group", bson.D{{"sum", bson.D{{"$sum", bson.A{}}}}}}},
			},
			err: &mongo.CommandError{
				Code:    40237,
				Name:    "Location40237",
				Message: "The $sum accumulator is a unary operator",
			},
			altMessage: "The $sum accumulator is a unary operator",
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			require.NotNil(t, tc.pipeline, "pipeline must not be nil")
			require.NotNil(t, tc.err, "err must not be nil")

			ctx, collection := setup.Setup(t)

			res, err := collection.Aggregate(ctx, tc.pipeline)

			assert.Nil(t, res)
			AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
		})
	}
}

func TestAggregateProjectErrors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		pipeline bson.A // required, aggregation pipeline stages

		err        *mongo.CommandError // required
		altMessage string              // optional, alternative error message
		skip       string              // optional, skip test with a specified reason
	}{
		"EmptyPipeline": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{}}},
			},
			err: &mongo.CommandError{
				Code:    51272,
				Name:    "Location51272",
				Message: "Invalid $project :: caused by :: projection specification must have at least one field",
			},
		},
		"EmptyProjectionField": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v", bson.D{}}}}},
			},
			err: &mongo.CommandError{
				Code:    51270,
				Name:    "Location51270",
				Message: "Invalid $project :: caused by :: An empty sub-projection is not a valid value. Found empty object at path",
			},
			skip: "https://github.com/FerretDB/FerretDB/issues/2633",
		},
		"EmptyKey": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"", true}}}},
			},
			err: &mongo.CommandError{
				Code: 40352,
				Name: "Location40352",
				Message: "Invalid $project :: caused by :: " +
					"FieldPath cannot be constructed with empty string",
			},
		},
		"EmptyPath": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v..d", true}}}},
			},
			err: &mongo.CommandError{
				Code:    15998,
				Name:    "Location15998",
				Message: "Invalid $project :: caused by :: FieldPath field names may not be empty strings.",
			},
		},
		"ExcludeInclude": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"foo", false}, {"bar", true}}}},
			},
			err: &mongo.CommandError{
				Code:    31253,
				Name:    "Location31253",
				Message: "Invalid $project :: caused by :: Cannot do inclusion on field bar in exclusion projection",
			},
			altMessage: "Cannot do inclusion on field bar in exclusion projection",
		},
		"IncludeExclude": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"foo", true}, {"bar", false}}}},
			},
			err: &mongo.CommandError{
				Code:    31254,
				Name:    "Location31254",
				Message: "Invalid $project :: caused by :: Cannot do exclusion on field bar in inclusion projection",
			},
			altMessage: "Cannot do exclusion on field bar in inclusion projection",
		},
		"PositionalOperatorMultiple": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v.$.foo.$", true}}}},
			},
			err: &mongo.CommandError{
				Code: 31324,
				Name: "Location31324",
				Message: "Invalid $project :: caused by :: " +
					"Cannot use positional projection in aggregation projection",
			},
		},
		"PositionalOperatorMiddle": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v.$.foo", true}}}},
			},
			err: &mongo.CommandError{
				Code: 31394,
				Name: "Location31394",
				Message: "Invalid $project :: caused by :: " +
					"As of 4.4, it's illegal to specify positional operator " +
					"in the middle of a path.Positional projection may only be " +
					"used at the end, for example: a.b.$. If the query previously " +
					"used a form like a.b.$.d, remove the parts following the '$' and " +
					"the results will be equivalent.",
			},
			altMessage: "Invalid $project :: caused by :: " +
				"Positional projection may only be used at the end, " +
				"for example: a.b.$. If the query previously used a form " +
				"like a.b.$.d, remove the parts following the '$' and " +
				"the results will be equivalent.",
		},
		"PositionalOperatorWrongLocations": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"$.v.$.foo", true}}}},
			},
			err: &mongo.CommandError{
				Code: 31394,
				Name: "Location31394",
				Message: "Invalid $project :: caused by :: " +
					"As of 4.4, it's illegal to specify positional operator " +
					"in the middle of a path.Positional projection may only be " +
					"used at the end, for example: a.b.$. If the query previously " +
					"used a form like a.b.$.d, remove the parts following the '$' and " +
					"the results will be equivalent.",
			},
			altMessage: "Invalid $project :: caused by :: " +
				"Positional projection may only be used at the end, " +
				"for example: a.b.$. If the query previously used a form " +
				"like a.b.$.d, remove the parts following the '$' and " +
				"the results will be equivalent.",
		},
		"PositionalOperatorEmptyPath": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v..$", true}}}},
			},
			err: &mongo.CommandError{
				Code: 31324,
				Name: "Location31324",
				Message: "Invalid $project :: caused by :: " +
					"Cannot use positional projection in aggregation projection",
			},
		},
		"PositionalOperatorDollarKey": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"$", true}}}},
			},
			err: &mongo.CommandError{
				Code: 16410,
				Name: "Location16410",
				Message: "Invalid $project :: caused by :: " +
					"FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
		"PositionalOperatorDollarInKey": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"$v", true}}}},
			},
			err: &mongo.CommandError{
				Code: 16410,
				Name: "Location16410",
				Message: "Invalid $project :: caused by :: " +
					"FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
		"PositionalOperatorDollarPrefix": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"$.foo", true}}}},
			},
			err: &mongo.CommandError{
				Code: 16410,
				Name: "Location16410",
				Message: "Invalid $project :: caused by :: " +
					"FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
		"PositionalOperatorDotDollarInKey": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v.$foo", true}}}},
			},
			err: &mongo.CommandError{
				Code: 16410,
				Name: "Location16410",
				Message: "Invalid $project :: caused by :: " +
					"FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
		"PositionalOperatorPrefixSuffix": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"$.v.$", true}}}},
			},
			err: &mongo.CommandError{
				Code: 31324,
				Name: "Location31324",
				Message: "Invalid $project :: caused by :: " +
					"Cannot use positional projection in aggregation projection",
			},
		},
		"PositionalOperatorExclusion": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v.$", false}}}},
			},
			err: &mongo.CommandError{
				Code: 31324,
				Name: "Location31324",
				Message: "Invalid $project :: caused by :: " +
					"Cannot use positional projection in aggregation projection",
			},
		},
		"ProjectPositionalOperator": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v.$", true}}}},
			},
			err: &mongo.CommandError{
				Code:    31324,
				Name:    "Location31324",
				Message: "Invalid $project :: caused by :: Cannot use positional projection in aggregation projection",
			},
		},
		"ProjectTypeEmpty": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v", bson.D{}}}}},
			},
			err: &mongo.CommandError{
				Code:    51270,
				Name:    "Location51270",
				Message: "Invalid $project :: caused by :: An empty sub-projection is not a valid value." + " Found empty object at path",
			},
		},
		"ProjectTwoOperators": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v", bson.D{{"$type", int32(42)}, {"$op", int32(42)}}}}}},
			},
			err: &mongo.CommandError{
				Code:    16410,
				Name:    "Location16410",
				Message: "Invalid $project :: caused by :: FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
		"ProjectTypeInvalidLen": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v", bson.D{{"$type", bson.A{"foo", "bar"}}}}}}},
			},
			err: &mongo.CommandError{
				Code:    16020,
				Name:    "Location16020",
				Message: "Invalid $project :: caused by :: Expression $type takes exactly 1 arguments. 2 were passed in.",
			},
		},
		"ProjectNonExistentOperator": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v", bson.D{{"$non-existent", "foo"}}}}}},
			},
			altMessage: "Invalid $project :: caused by :: Unrecognized expression '$non-existent'",
			err: &mongo.CommandError{
				Code:    31325,
				Name:    "Location31325",
				Message: "Invalid $project :: caused by :: Unknown expression $non-existent",
			},
		},
		"ProjectRecursiveNonExistentOperator": {
			pipeline: bson.A{
				bson.D{{"$project", bson.D{{"v", bson.D{{"$type", bson.D{{"$non-existent", "foo"}}}}}}}},
			},
			err: &mongo.CommandError{
				Code:    168,
				Name:    "InvalidPipelineOperator",
				Message: "Invalid $project :: caused by :: Unrecognized expression '$non-existent'",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			require.NotNil(t, tc.pipeline, "pipeline must not be nil")
			require.NotNil(t, tc.err, "err must not be nil")

			ctx, collection := setup.Setup(t)

			_, err := collection.Aggregate(ctx, tc.pipeline)
			AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
		})
	}
}

func TestAggregateSetErrors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct { //nolint:vet // used for test only
		pipeline bson.A // required, aggregation pipeline stages

		err        *mongo.CommandError // required
		altMessage string              // optional, alternative error message
		skip       string              // optional, skip test with a specified reason
	}{
		"NotDocument": {
			pipeline: bson.A{
				bson.D{{"$set", "not-document"}},
			},
			err: &mongo.CommandError{
				Code:    40272,
				Name:    "Location40272",
				Message: "$set specification stage must be an object, got string",
			},
			altMessage: "$set specification stage must be an object, got string",
		},
		"InvalidFieldPath": {
			pipeline: bson.A{
				bson.D{{"$set", bson.D{{"$foo", "v"}}}},
			},
			err: &mongo.CommandError{
				Code:    16410,
				Name:    "Location16410",
				Message: "Invalid $set :: caused by :: FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			require.NotNil(t, tc.pipeline, "pipeline must not be nil")
			require.NotNil(t, tc.err, "err must not be nil")

			ctx, collection := setup.Setup(t)

			_, err := collection.Aggregate(ctx, tc.pipeline)
			AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
		})
	}
}

func TestAggregateUnsetErrors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct { //nolint:vet // used for test only
		pipeline bson.A // required, aggregation pipeline stages

		err        *mongo.CommandError // required
		altMessage string              // optional, alternative error message
		skip       string              // optional, skip test with a specified reason
	}{
		"EmptyString": {
			pipeline: bson.A{
				bson.D{{"$unset", ""}},
			},
			err: &mongo.CommandError{
				Code:    40352,
				Name:    "Location40352",
				Message: "Invalid $unset :: caused by :: FieldPath cannot be constructed with empty string",
			},
		},
		"InvalidType": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.D{}}},
			},
			err: &mongo.CommandError{
				Code:    31002,
				Name:    "Location31002",
				Message: "$unset specification must be a string or an array",
			},
		},
		"PathEmptyKey": {
			pipeline: bson.A{
				bson.D{{"$unset", "v..foo"}},
			},
			err: &mongo.CommandError{
				Code:    15998,
				Name:    "Location15998",
				Message: "Invalid $unset :: caused by :: FieldPath field names may not be empty strings.",
			},
		},
		"PathEmptySuffixKey": {
			pipeline: bson.A{
				bson.D{{"$unset", "v."}},
			},
			err: &mongo.CommandError{
				Code:    40353,
				Name:    "Location40353",
				Message: "Invalid $unset :: caused by :: FieldPath must not end with a '.'.",
			},
		},
		"PathEmptyPrefixKey": {
			pipeline: bson.A{
				bson.D{{"$unset", ".v"}},
			},
			err: &mongo.CommandError{
				Code:    15998,
				Name:    "Location15998",
				Message: "Invalid $unset :: caused by :: FieldPath field names may not be empty strings.",
			},
		},
		"PathDollarPrefix": {
			pipeline: bson.A{
				bson.D{{"$unset", "$v"}},
			},
			err: &mongo.CommandError{
				Code:    16410,
				Name:    "Location16410",
				Message: "Invalid $unset :: caused by :: FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
		"ArrayEmpty": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{}}},
			},
			err: &mongo.CommandError{
				Code:    31119,
				Name:    "Location31119",
				Message: "$unset specification must be a string or an array with at least one field",
			},
			altMessage: "$unset specification must be a string or an array with at least one field",
		},
		"ArrayInvalidType": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"field1", 1}}},
			},
			err: &mongo.CommandError{
				Code:    31120,
				Name:    "Location31120",
				Message: "$unset specification must be a string or an array containing only string values",
			},
		},
		"ArrayEmptyString": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{""}}},
			},
			err: &mongo.CommandError{
				Code:    40352,
				Name:    "Location40352",
				Message: "Invalid $unset :: caused by :: FieldPath cannot be constructed with empty string",
			},
		},
		"ArrayPathDuplicate": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"v", "v"}}},
			},
			err: &mongo.CommandError{
				Code:    31250,
				Name:    "Location31250",
				Message: "Invalid $unset :: caused by :: Path collision at v",
			},
		},
		"ArrayPathOverwrites": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"v", "v.foo"}}},
			},
			err: &mongo.CommandError{
				Code:    31249,
				Name:    "Location31249",
				Message: "Invalid $unset :: caused by :: Path collision at v.foo remaining portion foo",
			},
		},
		"ArrayPathOverwritesRemaining": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"v", "v.foo.bar"}}},
			},
			err: &mongo.CommandError{
				Code:    31249,
				Name:    "Location31249",
				Message: "Invalid $unset :: caused by :: Path collision at v.foo.bar remaining portion foo.bar",
			},
		},
		"ArrayPathCollision": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"v.foo", "v"}}},
			},
			err: &mongo.CommandError{
				Code:    31250,
				Name:    "Location31250",
				Message: "Invalid $unset :: caused by :: Path collision at v",
			},
		},
		"ArrayPathEmptyKey": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"v..foo"}}},
			},
			err: &mongo.CommandError{
				Code:    15998,
				Name:    "Location15998",
				Message: "Invalid $unset :: caused by :: FieldPath field names may not be empty strings.",
			},
		},
		"ArrayPathEmptySuffixKey": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"v."}}},
			},
			err: &mongo.CommandError{
				Code:    40353,
				Name:    "Location40353",
				Message: "Invalid $unset :: caused by :: FieldPath must not end with a '.'.",
			},
		},
		"ArrayPathEmptyPrefixKey": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{".v"}}},
			},
			err: &mongo.CommandError{
				Code:    15998,
				Name:    "Location15998",
				Message: "Invalid $unset :: caused by :: FieldPath field names may not be empty strings.",
			},
		},
		"ArrayPathDollarPrefix": {
			pipeline: bson.A{
				bson.D{{"$unset", bson.A{"$v"}}},
			},
			err: &mongo.CommandError{
				Code:    16410,
				Name:    "Location16410",
				Message: "Invalid $unset :: caused by :: FieldPath field names may not start with '$'. Consider using $getField or $setField.",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			require.NotNil(t, tc.pipeline, "pipeline must not be nil")
			require.NotNil(t, tc.err, "err must not be nil")

			ctx, collection := setup.Setup(t)

			_, err := collection.Aggregate(ctx, tc.pipeline)
			AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
		})
	}
}
