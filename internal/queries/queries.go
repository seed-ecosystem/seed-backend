package queries

const InsertMsgQuery = `INSERT INTO messages (nonce, chat_id, signature, content, content_iv) VALUES ($1, $2, $3, $4, $5)`
const FetchHistoryQuery = `SELECT nonce, chat_id, signature, content, content_iv FROM messages WHERE chat_id = $1 AND nonce >= $2 ORDER BY nonce ASC LIMIT $3`
const LastNonceQuery = `SELECT MAX(nonce) FROM messages WHERE chat_id = $1`
