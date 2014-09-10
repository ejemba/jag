javap ../tests/java_example/out/production/java_example/local/Bar.class  | go run ../cmd/jagen/jagen.go  -src ../tests/java_example/src/local/Bar.java  > bar.go && javap ../tests/java_example/out/production/java_example/local/Foo.class  | go run ../cmd/jagen/jagen.go  -src ../tests/java_example/src/local/Foo.java  > foo.go && go build


