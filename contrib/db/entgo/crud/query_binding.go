package crud

import (
	"fmt"
	"math"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	crudpb "github.com/Servora-Kit/servora/api/gen/go/servora/crud/v1"
	corecrud "github.com/Servora-Kit/servora/core/crud"
)

func (fields *ListFields[PO]) filterPredicate(
	selector *sql.Selector,
	expression corecrud.FilterExpression,
) (*sql.Predicate, error) {
	if selector == nil {
		return nil, internalAdapterError("filter", "Ent selector is nil")
	}
	factory, err := fields.resolveFilter(expression)
	if err != nil || factory == nil {
		return nil, err
	}
	predicate := factory(selector)
	if predicate == nil {
		return nil, internalAdapterError("filter", "resolved predicate returned nil")
	}
	return predicate, nil
}

func (fields *ListFields[PO]) resolveFilter(expression corecrud.FilterExpression) (SelectorPredicate, error) {
	if fields == nil {
		return nil, internalAdapterError("filter", "ListFields is nil")
	}
	if expression.Empty() {
		return nil, nil
	}
	return fields.resolveFilterNode(expression.Root())
}

func (fields *ListFields[PO]) resolveFilterNode(node corecrud.FilterNode) (SelectorPredicate, error) {
	switch node.Kind() {
	case corecrud.FilterNodeAnd, corecrud.FilterNodeOr:
		children := node.Children()
		factories := make([]SelectorPredicate, 0, len(children))
		for _, child := range children {
			factory, err := fields.resolveFilterNode(child)
			if err != nil {
				return nil, err
			}
			if factory == nil {
				return nil, internalAdapterError("filter", "logical node resolved to nil")
			}
			factories = append(factories, factory)
		}
		isAnd := node.Kind() == corecrud.FilterNodeAnd
		return func(selector *sql.Selector) *sql.Predicate {
			predicates := make([]*sql.Predicate, len(factories))
			for index, factory := range factories {
				predicates[index] = factory(selector)
			}
			if isAnd {
				return sql.And(predicates...)
			}
			return sql.Or(predicates...)
		}, nil
	case corecrud.FilterNodeComparison, corecrud.FilterNodeHas:
		return fields.resolveFieldPredicate(node)
	default:
		return nil, internalAdapterError("filter", "unsupported typed AST node %d", node.Kind())
	}
}

func (fields *ListFields[PO]) resolveFieldPredicate(node corecrud.FilterNode) (SelectorPredicate, error) {
	field, ok := node.Field()
	if !ok {
		return nil, internalAdapterError("filter", "typed AST field is absent")
	}
	binding, ok := fields.bindings[field.Path()]
	if !ok || !binding.filter || binding.descriptor.FullName() != field.Descriptor().FullName() {
		return nil, invalidFilterError("field %q is not enabled by the repository", field.Path())
	}
	if node.Kind() == corecrud.FilterNodeHas && !field.Descriptor().IsList() {
		return nil, internalAdapterError("filter", "field %q has a non-repeated has node", field.Path())
	}

	value := node.Value()
	if value.Kind() == corecrud.FilterValueNull {
		if !binding.nullable {
			return nil, invalidFilterError("field %q is not nullable in the repository", field.Path())
		}
		return binding.resolveNullPredicate(node.Operator(), value)
	}
	var stringMatch corecrud.StringMatch
	isStringValue := value.Kind() == corecrud.FilterValueString
	if isStringValue {
		var ok bool
		stringMatch, ok = value.StringMatch()
		if !ok || stringMatch.Kind() == corecrud.StringMatchInvalid {
			return nil, internalAdapterError("filter", "field %q received an invalid StringMatch", field.Path())
		}
	}
	if binding.customPredicate != nil {
		factory, err := binding.customPredicate(node.Operator(), value)
		if err != nil {
			return nil, invalidFilterError("field %q custom predicate: %v", field.Path(), err)
		}
		if factory == nil {
			return nil, internalAdapterError("filter", "field %q custom predicate returned nil", field.Path())
		}
		return factory, nil
	}

	argument, err := binding.filterArgument(value)
	if err != nil {
		return nil, invalidFilterError("field %q: %v", field.Path(), err)
	}
	if node.Kind() == corecrud.FilterNodeHas {
		if binding.kind != bindingJSONPath {
			return nil, invalidFilterError("field %q has requires JSONPath or Custom binding", field.Path())
		}
		return func(*sql.Selector) *sql.Predicate {
			return sqljson.ValueContains(binding.column, argument, binding.jsonOptions()...)
		}, nil
	}
	if isStringValue {
		literal, ok := argument.(string)
		if ok {
			return binding.resolveStringPredicate(node.Operator(), stringMatch.Kind(), literal)
		}
		if stringMatch.Kind() != corecrud.StringMatchExact {
			return nil, internalAdapterError(
				"filter",
				"field %q converter returned %T for StringMatch kind %d",
				field.Path(), argument, stringMatch.Kind(),
			)
		}
	}
	switch binding.kind {
	case bindingColumn:
		return func(selector *sql.Selector) *sql.Predicate {
			predicate, _ := columnComparison(selector.C(binding.column), node.Operator(), argument, false)
			return predicate
		}, nil
	case bindingJSONPath:
		predicate, err := jsonComparison(binding.column, binding.jsonOptions(), node.Operator(), argument)
		if err != nil {
			return nil, err
		}
		return func(*sql.Selector) *sql.Predicate { return predicate }, nil
	default:
		return nil, internalAdapterError("filter", "field %q custom binding has no predicate", field.Path())
	}
}

func (binding fieldBinding) filterArgument(value corecrud.FilterValue) (any, error) {
	if binding.queryConverter != nil {
		return binding.queryConverter(value)
	}
	switch value.Kind() {
	case corecrud.FilterValueString:
		match, ok := value.StringMatch()
		if !ok {
			return nil, fmt.Errorf("invalid StringMatch")
		}
		return match.Literal(), nil
	case corecrud.FilterValueBool:
		result, _ := value.BoolValue()
		return result, nil
	case corecrud.FilterValueEnum:
		result, _ := value.EnumNumber()
		return int64(result), nil
	case corecrud.FilterValueInt64:
		result, _ := value.Int64Value()
		return result, nil
	case corecrud.FilterValueUint64:
		result, _ := value.Uint64Value()
		return result, nil
	case corecrud.FilterValueDouble:
		result, _ := value.DoubleValue()
		if math.IsNaN(result) || math.IsInf(result, 0) {
			return nil, fmt.Errorf("non-finite floating-point value")
		}
		return result, nil
	case corecrud.FilterValueTimestamp:
		result, _ := value.TimestampValue()
		return result, nil
	case corecrud.FilterValueDuration:
		result, _ := value.DurationValue()
		return time.Duration(result.AsDuration()), nil
	default:
		return nil, fmt.Errorf("unsupported typed literal kind %d", value.Kind())
	}
}

func (binding fieldBinding) resolveNullPredicate(
	operator corecrud.FilterOperator,
	value corecrud.FilterValue,
) (SelectorPredicate, error) {
	if operator != corecrud.FilterOperatorEqual && operator != corecrud.FilterOperatorNotEqual {
		return nil, invalidFilterError("NULL only supports = and !=")
	}
	if binding.kind == bindingCustom {
		factory, err := binding.customPredicate(operator, value)
		if err != nil {
			return nil, invalidFilterError("custom NULL predicate: %v", err)
		}
		if factory == nil {
			return nil, internalAdapterError("filter", "custom NULL predicate returned nil")
		}
		return factory, nil
	}
	if binding.kind == bindingJSONPath {
		expression := jsonStringExpression(binding.column, binding.jsonOptions())
		return func(*sql.Selector) *sql.Predicate {
			return sql.P(func(builder *sql.Builder) {
				expression(builder)
				if operator == corecrud.FilterOperatorEqual {
					builder.WriteString(" IS NULL")
					return
				}
				builder.WriteString(" IS NOT NULL")
			})
		}, nil
	}
	if operator == corecrud.FilterOperatorEqual {
		return func(selector *sql.Selector) *sql.Predicate {
			return sql.IsNull(selector.C(binding.column))
		}, nil
	}
	return func(selector *sql.Selector) *sql.Predicate {
		return sql.NotNull(selector.C(binding.column))
	}, nil
}

func (binding fieldBinding) jsonOptions() []sqljson.Option {
	return []sqljson.Option{sqljson.Path(binding.jsonPath...)}
}

type stringSQLExpression func(*sql.Builder)

func (binding fieldBinding) resolveStringPredicate(
	operator corecrud.FilterOperator,
	kind corecrud.StringMatchKind,
	literal string,
) (SelectorPredicate, error) {
	if err := validateStringMatchComparison(operator, kind); err != nil {
		return nil, err
	}
	switch binding.kind {
	case bindingColumn:
		return func(selector *sql.Selector) *sql.Predicate {
			predicate, _ := stringMatchComparison(
				columnStringExpression(selector.C(binding.column)), operator, kind, literal,
			)
			return predicate
		}, nil
	case bindingJSONPath:
		predicate, err := stringMatchComparison(
			jsonStringExpression(binding.column, binding.jsonOptions()), operator, kind, literal,
		)
		if err != nil {
			return nil, err
		}
		return func(*sql.Selector) *sql.Predicate { return predicate }, nil
	default:
		return nil, internalAdapterError("filter", "string match requires a column or JSONPath binding")
	}
}

func columnStringExpression(column string) stringSQLExpression {
	return func(builder *sql.Builder) {
		builder.Ident(column)
	}
}

func jsonStringExpression(column string, options []sqljson.Option) stringSQLExpression {
	textOptions := make([]sqljson.Option, 0, len(options)+1)
	textOptions = append(textOptions, sqljson.Unquote(true))
	textOptions = append(textOptions, options...)
	rawExpression := sqljson.ValuePath(column, options...)
	textExpression := sqljson.ValuePath(column, textOptions...)
	return func(builder *sql.Builder) {
		if builder.Dialect() == dialect.MySQL {
			builder.WriteString("CASE WHEN JSON_TYPE(")
			builder.Join(rawExpression)
			builder.WriteString(") = 'NULL' THEN NULL ELSE ")
			builder.Join(textExpression)
			builder.WriteString(" END")
			return
		}
		builder.Join(textExpression)
	}
}

func stringMatchComparison(
	expression stringSQLExpression,
	operator corecrud.FilterOperator,
	kind corecrud.StringMatchKind,
	literal string,
) (*sql.Predicate, error) {
	if err := validateStringMatchComparison(operator, kind); err != nil {
		return nil, err
	}
	if expression == nil {
		return nil, internalAdapterError("filter", "string SQL expression is nil")
	}
	if kind == corecrud.StringMatchExact {
		op, err := comparisonOp(operator)
		if err != nil {
			return nil, err
		}
		return sql.P(func(builder *sql.Builder) {
			writeBinaryStringExpression(builder, expression)
			builder.WriteOp(op).Arg(literal)
		}), nil
	}
	return sql.P(func(builder *sql.Builder) {
		if literal == "" && kind == corecrud.StringMatchContains && operator == corecrud.FilterOperatorEqual {
			writeBinaryStringExpression(builder, expression)
			builder.WriteString(" IS NOT NULL")
			return
		}
		negated := operator == corecrud.FilterOperatorNotEqual
		if negated {
			builder.WriteString("NOT (")
		}
		switch builder.Dialect() {
		case dialect.Postgres, dialect.MySQL:
			writeBinaryStringExpression(builder, expression)
			builder.WriteString(" LIKE ").Arg(escapedLikePattern(kind, literal)).WriteString(" ESCAPE '!'")
		default:
			writeSQLiteStringMatch(builder, expression, kind, literal)
		}
		if negated {
			builder.WriteByte(')')
		}
	}), nil
}

func validateStringMatchComparison(operator corecrud.FilterOperator, kind corecrud.StringMatchKind) error {
	if kind == corecrud.StringMatchInvalid {
		return internalAdapterError("filter", "invalid StringMatch kind")
	}
	if kind != corecrud.StringMatchExact &&
		operator != corecrud.FilterOperatorEqual && operator != corecrud.FilterOperatorNotEqual {
		return internalAdapterError("filter", "StringMatch kind %d uses operator %q", kind, operator)
	}
	if kind == corecrud.StringMatchExact {
		if _, err := comparisonOp(operator); err != nil {
			return internalAdapterError("filter", "exact StringMatch uses operator %q", operator)
		}
	}
	return nil
}

func escapedLikePattern(kind corecrud.StringMatchKind, literal string) string {
	var pattern strings.Builder
	pattern.Grow(len(literal) + 2)
	if kind == corecrud.StringMatchSuffix || kind == corecrud.StringMatchContains {
		pattern.WriteByte('%')
	}
	for index := range len(literal) {
		current := literal[index]
		if current == '!' || current == '%' || current == '_' {
			pattern.WriteByte('!')
		}
		pattern.WriteByte(current)
	}
	if kind == corecrud.StringMatchPrefix || kind == corecrud.StringMatchContains {
		pattern.WriteByte('%')
	}
	return pattern.String()
}

func writeSQLiteStringMatch(
	builder *sql.Builder,
	expression stringSQLExpression,
	kind corecrud.StringMatchKind,
	literal string,
) {
	switch kind {
	case corecrud.StringMatchPrefix, corecrud.StringMatchContains:
		builder.WriteString("instr(")
		writeBinaryStringExpression(builder, expression)
		builder.Comma().Arg(literal).WriteByte(')')
		if kind == corecrud.StringMatchPrefix {
			builder.WriteString(" = 1")
		} else {
			builder.WriteString(" > 0")
		}
	case corecrud.StringMatchSuffix:
		builder.WriteString("substr(")
		expression(builder)
		builder.WriteString(", -length(").Arg(literal).WriteString(")) COLLATE BINARY = ").Arg(literal)
	}
}

func columnComparison(
	column string,
	operator corecrud.FilterOperator,
	argument any,
	binaryString bool,
) (*sql.Predicate, error) {
	if binaryString {
		return binaryStringComparison(column, operator, argument)
	}
	switch operator {
	case corecrud.FilterOperatorEqual:
		return sql.EQ(column, argument), nil
	case corecrud.FilterOperatorNotEqual:
		return sql.NEQ(column, argument), nil
	case corecrud.FilterOperatorLess:
		return sql.LT(column, argument), nil
	case corecrud.FilterOperatorLessEqual:
		return sql.LTE(column, argument), nil
	case corecrud.FilterOperatorGreater:
		return sql.GT(column, argument), nil
	case corecrud.FilterOperatorGreaterEqual:
		return sql.GTE(column, argument), nil
	default:
		return nil, invalidFilterError("operator %q is unsupported by an ordinary column", operator)
	}
}

func binaryStringComparison(
	column string,
	operator corecrud.FilterOperator,
	argument any,
) (*sql.Predicate, error) {
	op, err := comparisonOp(operator)
	if err != nil {
		return nil, err
	}
	return sql.P(func(builder *sql.Builder) {
		writeBinaryStringExpression(builder, columnStringExpression(column))
		builder.WriteOp(op).Arg(argument)
	}), nil
}

func writeBinaryStringExpression(builder *sql.Builder, expression stringSQLExpression) {
	switch builder.Dialect() {
	case dialect.MySQL:
		builder.WriteString("BINARY ")
		expression(builder)
	case dialect.Postgres:
		expression(builder)
		builder.WriteString(` COLLATE "C"`)
	default:
		expression(builder)
		builder.WriteString(" COLLATE BINARY")
	}
}

func comparisonOp(operator corecrud.FilterOperator) (sql.Op, error) {
	switch operator {
	case corecrud.FilterOperatorEqual:
		return sql.OpEQ, nil
	case corecrud.FilterOperatorNotEqual:
		return sql.OpNEQ, nil
	case corecrud.FilterOperatorLess:
		return sql.OpLT, nil
	case corecrud.FilterOperatorLessEqual:
		return sql.OpLTE, nil
	case corecrud.FilterOperatorGreater:
		return sql.OpGT, nil
	case corecrud.FilterOperatorGreaterEqual:
		return sql.OpGTE, nil
	default:
		return 0, invalidFilterError("operator %q is unsupported", operator)
	}
}

func jsonComparison(
	column string,
	options []sqljson.Option,
	operator corecrud.FilterOperator,
	argument any,
) (*sql.Predicate, error) {
	switch operator {
	case corecrud.FilterOperatorEqual:
		return sqljson.ValueEQ(column, argument, options...), nil
	case corecrud.FilterOperatorNotEqual:
		return sqljson.ValueNEQ(column, argument, options...), nil
	case corecrud.FilterOperatorLess:
		return sqljson.ValueLT(column, argument, options...), nil
	case corecrud.FilterOperatorLessEqual:
		return sqljson.ValueLTE(column, argument, options...), nil
	case corecrud.FilterOperatorGreater:
		return sqljson.ValueGT(column, argument, options...), nil
	case corecrud.FilterOperatorGreaterEqual:
		return sqljson.ValueGTE(column, argument, options...), nil
	default:
		return nil, invalidFilterError("operator %q is unsupported by JSONPath", operator)
	}
}

func invalidFilterError(format string, args ...any) error {
	return crudpb.ErrorCrudErrorReasonInvalidFilter("%s", "filter: "+fmt.Sprintf(format, args...))
}

func internalAdapterError(path, format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	if path != "" {
		message = path + ": " + message
	}
	return crudpb.ErrorCrudErrorReasonInternal("%s", message)
}
