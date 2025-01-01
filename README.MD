Note: This repo will be empty until I get the snail-to-cheetah basic implmentation of this project done. (serail fullread) It is this tool, it's just cut down -- no flags because I want to focus on the implementation method - serial, v. parallel, full-read v. streaming, etc. 

Personal Notes:
- This is a also good opportunity to understand how autocomplete works with flags and different terminals
   - See the autocomplete repo (currently private) 
- This will be written in Go

## Description
This tool will check the front matter for markdown files to ensure they align to a specified format.

**IMPORTANT: It is ASSUMED that the files are part of a version control repository (z.b. git). This tool DOES NOT backup files before making changes to them. There is however a log of chnges made, which can be found in <TBD>. This tool does not automatically add or commit changes to the repo, that remains the task of the end user**

## Required Flags:

### Template
- -t | --template <path to template>
	- Will return an error if the template does not exist

### Analyze or Fix 
- analyze 
	- Will only provide an output. For now this output is of this format:
	
	| FullPath | <Prop Name (From Template)> | <Prop Name (From Template)> | <Prop Name(Not in Template)> | 
	|---|---| --- | -- |
	| <-filename with full **project** path->  | <-missing \| present(no value) \| hasValue -> | <-missing \| present(no value) \| hasValue ->| <-missing \| present(no value) \| hasValue ->
	| <-filename with full **project** path->  | <-missing \| present(no value) \| hasValue -> | <-missing \| present(no value) \| hasValue ->| <-missing \| present(no value) \| hasValue ->

**The prop names specified in the template will always appear before the prop names that exist in the files but do NOT exist in the template.**

- fix
	- Will fix issues. You must also add more flags to specify the type of fix:
	
#### Fix Options:
When using the fix flag we must add additional flags to determine exactly the modifications to be made.

- fullConform
	- Ensures that the front matter fully conforms to the specification. (Includes all properties and they are in the specified order). 
		- This option **does** ensure that the front matter starts on the first line of the file
		- This option **does** remove extra properties (& the values they may hold) 
		- This option **does** generate IDs if ID is a property specified in the template
		- This option **does not** check that the properties contain values, only that the properties exist. 

- allProps
	- This option ensures that all the properties in the template exist, 

- fixOrder
	- Reorders the properties to match the template specification

- removeExtraProps
    - This will only remove properties not defined in the template

#### Files to Be Analyzed/Fixed
It is also required to use the `-dir` flag or the `-files` flag. This tells the script which files/directories we want to fix or analyze.

-  dir 
	- A single directory, relative to the project root. An error will be returned in the directory doesn't exist

- files
	- One or many files
		- Note for Dev: this wilkl essentially just skip the dir traversal step and check for these files directly (and add the to the filePathsToProcess array) 

#### Optional Flags
Value Insertion:
- - genID
	- Generate an ID if ID is a property specified in the template and the ID property is either missing or has no value. 
- vfc | -- keyValuesFromConfig
	- Uses the key value pairs from the valueInsertion property of the config json
		- Implementation note. Must be used with the -config flag
- help 
	- Display this file


## Output
The program emits the template to the console as well as an explanation of the flags chosen and asks you if you want to proceed [Y/N]
 - This check can be skipped with the `noCheck` setting set to true in the config file. 

## Config File

The config file can be used to 

`-config <absolute filesystem path>`

### Format

```json
{
	valueInsertion:
	{
		prop: "value to insert",
		anotherProp: "['value', 'value']"
	}
	
	
}
```
Note the order of the properties does not need to match the order of the properties in the template, nor do all the properties in the template need to appear here.


## Example Use: 
`fmc -t <template path> -analyze -dir ./myfolder/ `
Prints the output table to the console 

`fmc -t <template path> -fix -a -dir / `
This would fix every file in the project. All files would now exactly match the template, see -a notes for more info.

`fmc -t <template path> -vfc -fix -a -dir / -config`


 Later Stuff:
 -- Idea: What about chaining... e.g. if this prop exists with this value then create a new prop:value pair... 


