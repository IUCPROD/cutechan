delete from sessions
  where account = $1 and token = $2
