-- name: GetNote :one
SELECT * FROM notes WHERE id = $1 AND user_id = $2;

-- name: CreateNote :one
INSERT INTO notes(id,user_id,work_date,title,project,description,tasks,status,priority,tags,minutes,blockers,next_steps)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING *;

-- name: UpdateNote :one
UPDATE notes SET work_date=$3,title=$4,project=$5,description=$6,tasks=$7,status=$8,priority=$9,tags=$10,minutes=$11,blockers=$12,next_steps=$13,version=version+1,updated_at=now()
WHERE id=$1 AND user_id=$2 AND version=$14 RETURNING *;

-- name: DeleteNote :execrows
DELETE FROM notes WHERE id=$1 AND user_id=$2 AND version=$3;

-- name: ListNotes :many
SELECT * FROM notes WHERE user_id = sqlc.arg(owner)
AND (sqlc.arg(search)::text = '' OR (title || ' ' || description || ' ' || project || ' ' || array_to_string(tags,' ')) ILIKE '%' || sqlc.arg(search) || '%')
AND (sqlc.arg(from_date)::text = '' OR work_date >= NULLIF(sqlc.arg(from_date),'')::date)
AND (sqlc.arg(to_date)::text = '' OR work_date <= NULLIF(sqlc.arg(to_date),'')::date)
AND (sqlc.arg(project_filter)::text = '' OR project = sqlc.arg(project_filter))
AND (sqlc.arg(status_filter)::text = '' OR status = sqlc.arg(status_filter))
AND (sqlc.arg(priority_filter)::text = '' OR priority = sqlc.arg(priority_filter))
AND (sqlc.arg(tag_filter)::text = '' OR sqlc.arg(tag_filter) = ANY(tags))
ORDER BY
 CASE WHEN sqlc.arg(sort_by)::text = 'title' THEN title END ASC,
 CASE WHEN sqlc.arg(sort_by)::text = 'priority' THEN CASE priority WHEN 'high' THEN 3 WHEN 'medium' THEN 2 ELSE 1 END END DESC,
 CASE WHEN sqlc.arg(sort_by)::text = 'updated' THEN updated_at END DESC,
 work_date DESC,id
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);
