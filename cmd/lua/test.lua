

local ffi = require "ffi"


local mock = ffi.load("/amesh/amesh.so")

ffi.cdef[[
typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
typedef GoInt64 GoInt;
typedef GoUint64 GoUint;
typedef float GoFloat32;
typedef double GoFloat64;
typedef float _Complex GoComplex64;
typedef double _Complex GoComplex128;

typedef struct { const char *p; ptrdiff_t n; } _GoString_;
typedef _GoString_ GoString;

typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;


extern void CreateMock(void* writeRoute);
extern void MockForever(void* zone);
extern void Log(GoString msg);
]]


xdsSrc = ffi.new("GoString")
local s = "grpc://istiod.istio-system.svc.cluster.local:15010"
xdsSrc.p = s;
xdsSrc.n = #s;

