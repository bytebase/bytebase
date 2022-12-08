# https://developers.google.com/protocol-buffers/docs/gotutorial

# store package belongs to storage related proto's.
protoc --proto_path=store --go_out=. store/database.proto
