jag
===

Java Api Generator

Generate Go bindings for a Java API.

####Examples
* Amazon EC2 SDK for Go http://github.com/timob/ec2
* Cassandra DB monitoring API for Go http://github.com/timob/node_probe

####Using
Installing the packge creates the jagen command. This command will generate a Go file on standard output. It takes as input the output of the javap command (part of the JDK) for a given class file. The generated file will use the http://github.com/timob/javabind package to call the Java API through JNI.

There is example in tests/ directory. Which can be generated with gen.sh script. And there is a main program that uses generated code in cmd/.

#####Generated Code
A function is created for each constructor, it returns a pointer to a Go struct that has methods corresponding to the Java object. Basic Java types are converted and some common objects are converted by the javabind pacakge, List, Map, String etc... 

Constrcutors/Methods that can throw exceptions, have an error return value, which if non nil will represent the exception.

####Status
Not much testing has been done. I've generated a few APIs and successfully used them running on OpenJDK.

Todo:
* Make object conversion customizable/implementable outside the javabind package, currently object conversions are defined in the source code. Should be loaded from config file.
* Figure out how to handle Java Class methods that have the same name, (Go method names must be unique).
* Add javadoc comments to generated methods/constructors.
* More customizable way of shortening Java class names.

