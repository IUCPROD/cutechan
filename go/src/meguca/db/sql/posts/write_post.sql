insert into posts (
  editing, id, board, op, time, body, name, trip, auth, password, ip,
  SHA1, links, commands
)
  values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
