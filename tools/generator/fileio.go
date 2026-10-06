package generator

import "os"

// writeFile is shared by the remaining framework rendering compatibility paths
// so tests can inject write failures without changing their public APIs.
var writeFile = os.WriteFile
