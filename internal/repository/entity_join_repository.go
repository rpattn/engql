package repository

import (
	"context"
	"fmt"
	"strings"

	"graphql-engineering-api/internal/db"
	"graphql-engineering-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type entityJoinRepository struct {
	queries *db.Queries
	db      db.DBTX
}

// NewEntityJoinRepository creates a repository for managing join definitions
func NewEntityJoinRepository(queries *db.Queries, exec db.DBTX) EntityJoinRepository {
	return &entityJoinRepository{
		queries: queries,
		db:      exec,
	}
}

func (r *entityJoinRepository) Create(ctx context.Context, join domain.EntityJoinDefinition) (domain.EntityJoinDefinition, error) {
	leftFiltersJSON, err := domain.FiltersToJSONB(join.LeftFilters)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("marshal left filters: %w", err)
	}
	rightFiltersJSON, err := domain.FiltersToJSONB(join.RightFilters)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("marshal right filters: %w", err)
	}
	sortJSON, err := domain.SortCriteriaToJSONB(join.SortCriteria)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("marshal sort criteria: %w", err)
	}

	row, err := r.queries.CreateEntityJoin(ctx, db.CreateEntityJoinParams{
		OrganizationID:  join.OrganizationID,
		Name:            join.Name,
		Description:     pgtype.Text{String: join.Description, Valid: join.Description != ""},
		LeftEntityType:  join.LeftEntityType,
		RightEntityType: join.RightEntityType,
		JoinField:       join.JoinField,
		JoinFieldType:   string(join.JoinFieldType),
		LeftFilters:     leftFiltersJSON,
		RightFilters:    rightFiltersJSON,
		SortCriteria:    sortJSON,
	})
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("create entity join: %w", err)
	}

	return mapJoinRow(row)
}

func (r *entityJoinRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.EntityJoinDefinition, error) {
	row, err := r.queries.GetEntityJoin(ctx, id)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("get entity join: %w", err)
	}
	return mapJoinRow(row)
}

func (r *entityJoinRepository) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]domain.EntityJoinDefinition, error) {
	rows, err := r.queries.ListEntityJoinsByOrganization(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list entity joins: %w", err)
	}

	result := make([]domain.EntityJoinDefinition, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapJoinRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}

	return result, nil
}

func (r *entityJoinRepository) Update(ctx context.Context, join domain.EntityJoinDefinition) (domain.EntityJoinDefinition, error) {
	leftFiltersJSON, err := domain.FiltersToJSONB(join.LeftFilters)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("marshal left filters: %w", err)
	}
	rightFiltersJSON, err := domain.FiltersToJSONB(join.RightFilters)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("marshal right filters: %w", err)
	}
	sortJSON, err := domain.SortCriteriaToJSONB(join.SortCriteria)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("marshal sort criteria: %w", err)
	}

	row, err := r.queries.UpdateEntityJoin(ctx, db.UpdateEntityJoinParams{
		ID:              join.ID,
		Name:            join.Name,
		Description:     pgtype.Text{String: join.Description, Valid: join.Description != ""},
		LeftEntityType:  join.LeftEntityType,
		RightEntityType: join.RightEntityType,
		JoinField:       join.JoinField,
		JoinFieldType:   string(join.JoinFieldType),
		LeftFilters:     leftFiltersJSON,
		RightFilters:    rightFiltersJSON,
		SortCriteria:    sortJSON,
	})
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("update entity join: %w", err)
	}

	return mapJoinRow(row)
}

func (r *entityJoinRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteEntityJoin(ctx, id); err != nil {
		return fmt.Errorf("delete entity join: %w", err)
	}
	return nil
}

func (r *entityJoinRepository) ExecuteJoin(ctx context.Context, join domain.EntityJoinDefinition, options domain.JoinExecutionOptions) ([]domain.EntityJoinEdge, int64, error) {
	builder := newSQLBuilder()

	leftAlias := "l"
	rightAlias := "r"

	joinFieldIdx := builder.addArg(join.JoinField)
	orgIdx := builder.addArg(join.OrganizationID)
	leftTypeIdx := builder.addArg(join.LeftEntityType)
	rightTypeIdx := builder.addArg(join.RightEntityType)

	var fromBuilder strings.Builder
	fromBuilder.WriteString("FROM entities ")
	fromBuilder.WriteString(leftAlias)
	fromBuilder.WriteString(" ")

	switch join.JoinFieldType {
	case domain.FieldTypeEntityReferenceArray:
		fromBuilder.WriteString(fmt.Sprintf("JOIN LATERAL jsonb_array_elements_text(COALESCE("+
			"%s.properties -> %s::text, '[]'::jsonb)) AS jf(value) ON TRUE ", leftAlias, builder.placeholder(joinFieldIdx)))
		fromBuilder.WriteString(fmt.Sprintf("JOIN entities %s ON %s.id::text = jf.value ", rightAlias, rightAlias))
	default:
		fromBuilder.WriteString(fmt.Sprintf("JOIN entities %s ON %s.id::text = %s.properties ->> %s::text ",
			rightAlias, rightAlias, leftAlias, builder.placeholder(joinFieldIdx)))
	}

	whereClauses := []string{
		fmt.Sprintf("%s.organization_id = %s", leftAlias, builder.placeholder(orgIdx)),
		fmt.Sprintf("%s.organization_id = %s", rightAlias, builder.placeholder(orgIdx)),
		fmt.Sprintf("%s.entity_type = %s", leftAlias, builder.placeholder(leftTypeIdx)),
		fmt.Sprintf("%s.entity_type = %s", rightAlias, builder.placeholder(rightTypeIdx)),
	}

	leftFilters := append([]domain.JoinPropertyFilter{}, join.LeftFilters...)
	if len(options.LeftFilters) > 0 {
		leftFilters = append(leftFilters, options.LeftFilters...)
	}
	rightFilters := append([]domain.JoinPropertyFilter{}, join.RightFilters...)
	if len(options.RightFilters) > 0 {
		rightFilters = append(rightFilters, options.RightFilters...)
	}

	for _, filter := range leftFilters {
		appendFilterClauses(leftAlias, filter, builder, &whereClauses)
	}

	for _, filter := range rightFilters {
		appendFilterClauses(rightAlias, filter, builder, &whereClauses)
	}

	if len(whereClauses) > 0 {
		fromBuilder.WriteString("WHERE ")
		fromBuilder.WriteString(strings.Join(whereClauses, " AND "))
		fromBuilder.WriteString(" ")
	}

	combinedSorts := append([]domain.JoinSortCriterion{}, join.SortCriteria...)
	if len(options.SortCriteria) > 0 {
		combinedSorts = append(combinedSorts, options.SortCriteria...)
	}

	countArgs := append([]any{}, builder.args...)

	orderClause := buildOrderClause(combinedSorts, builder, join, leftAlias, rightAlias, joinFieldIdx)

	selectClause := fmt.Sprintf("SELECT %s.id, %s.organization_id, %s.entity_type, %s.path, %s.properties, %s.created_at, %s.updated_at, "+
		"%s.id, %s.organization_id, %s.entity_type, %s.path, %s.properties, %s.created_at, %s.updated_at ",
		leftAlias, leftAlias, leftAlias, leftAlias, leftAlias, leftAlias, leftAlias,
		rightAlias, rightAlias, rightAlias, rightAlias, rightAlias, rightAlias, rightAlias)

	baseQuery := selectClause + fromBuilder.String()
	countQuery := "SELECT COUNT(*) " + fromBuilder.String()

	limit := options.Limit
	if limit <= 0 {
		limit = 25
	}
	offset := options.Offset
	if offset < 0 {
		offset = 0
	}

	limitIdx := builder.addArg(limit)
	offsetIdx := builder.addArg(offset)

	resultQuery := baseQuery
	if orderClause != "" {
		resultQuery += orderClause + " "
	}
	resultQuery += fmt.Sprintf("LIMIT %s OFFSET %s", builder.placeholder(limitIdx), builder.placeholder(offsetIdx))

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count join results: %w", err)
	}

	rows, err := r.db.Query(ctx, resultQuery, builder.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("execute join query: %w", err)
	}
	defer rows.Close()

	var edges []domain.EntityJoinEdge

	for rows.Next() {
		var (
			leftRow  db.Entity
			rightRow db.Entity
		)
		if err := rows.Scan(
			&leftRow.ID,
			&leftRow.OrganizationID,
			&leftRow.EntityType,
			&leftRow.Path,
			&leftRow.Properties,
			&leftRow.CreatedAt,
			&leftRow.UpdatedAt,
			&rightRow.ID,
			&rightRow.OrganizationID,
			&rightRow.EntityType,
			&rightRow.Path,
			&rightRow.Properties,
			&rightRow.CreatedAt,
			&rightRow.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan join row: %w", err)
		}

		leftEntity, err := mapDBEntity(leftRow)
		if err != nil {
			return nil, 0, err
		}
		rightEntity, err := mapDBEntity(rightRow)
		if err != nil {
			return nil, 0, err
		}

		edges = append(edges, domain.EntityJoinEdge{
			Left:  leftEntity,
			Right: rightEntity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate join rows: %w", err)
	}

	return edges, total, nil
}

func mapJoinRow(row db.EntityJoin) (domain.EntityJoinDefinition, error) {
	leftFilters, err := domain.FiltersFromJSONB(row.LeftFilters)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("decode left filters: %w", err)
	}
	rightFilters, err := domain.FiltersFromJSONB(row.RightFilters)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("decode right filters: %w", err)
	}
	sorts, err := domain.SortCriteriaFromJSONB(row.SortCriteria)
	if err != nil {
		return domain.EntityJoinDefinition{}, fmt.Errorf("decode sort criteria: %w", err)
	}

	description := ""
	if row.Description.Valid {
		description = row.Description.String
	}

	return domain.EntityJoinDefinition{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Name:            row.Name,
		Description:     description,
		LeftEntityType:  row.LeftEntityType,
		RightEntityType: row.RightEntityType,
		JoinField:       row.JoinField,
		JoinFieldType:   domain.FieldType(row.JoinFieldType),
		LeftFilters:     leftFilters,
		RightFilters:    rightFilters,
		SortCriteria:    sorts,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func mapDBEntity(row db.Entity) (domain.Entity, error) {
	properties, err := domain.FromJSONBProperties(row.Properties)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("decode entity properties: %w", err)
	}

	return domain.Entity{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		EntityType:     row.EntityType,
		Path:           row.Path,
		Properties:     properties,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

type sqlBuilder struct {
	args []any
}

func newSQLBuilder() *sqlBuilder {
	return &sqlBuilder{args: make([]any, 0)}
}

func (b *sqlBuilder) addArg(value any) int {
	b.args = append(b.args, value)
	return len(b.args)
}

func (b *sqlBuilder) placeholder(idx int) string {
	return fmt.Sprintf("$%d", idx)
}

func appendFilterClauses(alias string, filter domain.JoinPropertyFilter, builder *sqlBuilder, where *[]string) {
	if filter.Key == "" {
		return
	}

	keyIdx := builder.addArg(filter.Key)
	keyPlaceholder := builder.placeholder(keyIdx)

	if filter.Exists != nil {
		expr := fmt.Sprintf("%s.properties ? %s::text", alias, keyPlaceholder)
		if !*filter.Exists {
			expr = "NOT (" + expr + ")"
		}
		*where = append(*where, expr)
	}

	if filter.Value != nil {
		valIdx := builder.addArg(*filter.Value)
		*where = append(*where, fmt.Sprintf("%s.properties ->> %s::text = %s", alias, keyPlaceholder, builder.placeholder(valIdx)))
	}

	if len(filter.InArray) > 0 {
		arrIdx := builder.addArg(filter.InArray)
		clause := fmt.Sprintf("("+
			"%s.properties ->> %s::text = ANY(%s::text[]) OR "+
			"EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(%s.properties -> %s::text, '[]'::jsonb)) AS arr(val) "+
			"WHERE arr.val = ANY(%s::text[])))",
			alias, keyPlaceholder, builder.placeholder(arrIdx),
			alias, keyPlaceholder, builder.placeholder(arrIdx))
		*where = append(*where, clause)
	}
}

func buildOrderClause(sorts []domain.JoinSortCriterion, builder *sqlBuilder, join domain.EntityJoinDefinition, leftAlias, rightAlias string, joinFieldIdx int) string {
	if len(sorts) == 0 {
		return "ORDER BY " + leftAlias + ".created_at DESC"
	}

	orderings := make([]string, 0, len(sorts))
	for _, sort := range sorts {
		if sort.Field == "" {
			continue
		}
		direction := strings.ToUpper(string(sort.Direction))
		if direction != string(domain.JoinSortDesc) {
			direction = string(domain.JoinSortAsc)
		}

		targetAlias := leftAlias
		if strings.EqualFold(string(sort.Side), string(domain.JoinSideRight)) {
			targetAlias = rightAlias
		}

		orderExpr := buildSortExpression(targetAlias, sort.Field, join, builder, leftAlias, joinFieldIdx)
		if orderExpr == "" {
			continue
		}

		orderings = append(orderings, fmt.Sprintf("%s %s NULLS LAST", orderExpr, direction))
	}

	if len(orderings) == 0 {
		return "ORDER BY " + leftAlias + ".created_at DESC"
	}

	return "ORDER BY " + strings.Join(orderings, ", ")
}

func buildSortExpression(alias, field string, join domain.EntityJoinDefinition, builder *sqlBuilder, leftAlias string, joinFieldIdx int) string {
	switch strings.ToLower(field) {
	case "createdat":
		return alias + ".created_at"
	case "updatedat":
		return alias + ".updated_at"
	case "path":
		return alias + ".path"
	case "entitytype":
		return alias + ".entity_type"
	case "id":
		return alias + ".id::text"
	}

	if alias == leftAlias && strings.EqualFold(field, join.JoinField) {
		if join.JoinFieldType == domain.FieldTypeEntityReferenceArray {
			return "jf.value"
		}
		return fmt.Sprintf("%s.properties ->> %s::text", alias, builder.placeholder(joinFieldIdx))
	}

	fieldIdx := builder.addArg(field)
	return fmt.Sprintf("%s.properties ->> %s::text", alias, builder.placeholder(fieldIdx))
}
