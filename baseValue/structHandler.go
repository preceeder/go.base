package baseValue

import (
	"reflect"
	"strings"
)

type StructMap struct {
	Method map[string]reflect.Method
	Attr   map[string]reflect.Value
	Rcvr   reflect.Value
	Typ    reflect.Type
}

// MakeService rep 的 传入 指针 非指针都可以
// 解析对象的 方法 可以通过 map的形式执行
func MakeService(rep interface{}) *StructMap {
	ser := StructMap{}
	ser.Typ = reflect.TypeOf(rep)
	ser.Rcvr = reflect.ValueOf(rep)
	value := ser.Rcvr
	if value.Kind() == reflect.Ptr {
		value = ser.Rcvr.Elem()
	}

	ser.Attr = map[string]reflect.Value{}
	tp := ReflectBaseType(ser.Typ)

	for i := 0; i < tp.NumField(); i++ {
		val := value.Field(i)
		mname := tp.Field(i).Name // string
		ser.Attr[mname] = val
	}

	ser.Method = map[string]reflect.Method{}
	for i := 0; i < ser.Typ.NumMethod(); i++ {
		method := ser.Typ.Method(i)
		mname := method.Name // string
		ser.Method[mname] = method
	}

	return &ser
}

func StructConvertToStructWithTag(output any, tag string, weaklyTypedInput bool, input ...any) error {
	config := &DecoderConfig{
		Metadata:         nil,
		Result:           output,
		TagName:          tag,
		Squash:           true,             // 匿名结构体也会处理,  false 就不会处理匿名结构体中的数据
		WeaklyTypedInput: weaklyTypedInput, // []uint8  需要 处理为 string 类型 的就需要为true
	}
	decoder, err := NewDecoder(config)
	if err != nil {
		//slog.Error("mapToStruct mapstructure.NewDecoder(config)", "error", err.Error())
		return err
	}
	for _, it := range input {
		err := decoder.Decode(it)
		if err != nil {
			return err
		}
	}
	return nil
}

// StructSetDefaultValue 给对象中的数据设置默认值
// 可以处理匿名结构体
// obj 得是指针
// tag 标签名， 不一定是 default,  第一个值必须是需要填充的默认值，可以有第二个参数 'canZero' 可以为零值
// default 值 想要是 空字符串  得是后面的写法   default:"\"\""
func StructSetDefaultValue[T any](obj T, tag string) error {
	if tag == "" {
		tag = "default"
	}
	v := reflect.ValueOf(obj).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)
		// 私有属性不修改
		if structField.PkgPath != "" {
			continue
		}
		if field.Kind() == reflect.Struct {
			err := StructSetDefaultValue(field.Addr().Interface(), tag)
			if err != nil {
				return err
			}
		}
		defaultValue := structField.Tag.Get("default")
		if defaultValue == "" {
			continue
		}
		// 目前可以有两个值 第一个是默认值， 第二个是条件 canZero， 有这个表示可以为 '零值'
		defaultValues := strings.Split(defaultValue, ",")
		// 空指针需要处理
		if field.IsZero() {
			if defaultValues[0] != "" {
				newValue, err := ParseValueByType(field.Type(), defaultValues[0])
				if err != nil {
					return err
				}
				field.Set(newValue)
			} else if len(defaultValues) > 1 && defaultValues[1] == "canZero" {
				ft := field.Type()
				if ft.Kind() == reflect.Ptr {
					field.Set(reflect.New(ft.Elem()))
				} else {
					field.Set(reflect.Zero(ft))
				}
			}
		}
	}
	return nil
}

// StructGetTagValueNames 获取结构体指定的 tag valueName名， 目前用于提取结构体中db 标签的值
// obj 是不是指针都无所谓
func StructGetTagValueNames[T any](obj T, tag string) []string {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v)
	t := v.Type()
	var valueNames = make([]string, 0)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)
		// 私有属性跳过
		if structField.PkgPath != "" {
			continue
		}
		if field.Kind() == reflect.Struct && structField.Anonymous {
			if field.CanAddr() {
				subValueNames := StructGetTagValueNames(field.Addr().Interface(), tag)
				valueNames = append(valueNames, subValueNames...)
			} else {
				subValueNames := StructGetTagValueNames(field.Interface(), tag)
				valueNames = append(valueNames, subValueNames...)
			}
			continue
		}

		valueName := structField.Tag.Get(tag)
		if valueName == "" {
			continue
		}

		valueNames = append(valueNames, valueName)
	}
	return valueNames
}
