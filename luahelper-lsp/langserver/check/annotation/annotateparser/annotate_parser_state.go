package annotateparser

import (
	"luahelper-lsp/langserver/check/annotation/annotateast"
	"luahelper-lsp/langserver/check/annotation/annotatelexer"
	"luahelper-lsp/langserver/check/compiler/lexer"
)

// 解析最基础的@type
// ---@type MY_TYPE[|OTHER_TYPE] [@comment]
// ---@type MY_TYPE[|OTHER_TYPE], MY_TYPE[|OTHER_TYPE] [@comment] [@comment]
func parserTypeState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// skip type token
	l.NextTokenOfKind(annotatelexer.ATokenKwType)

	typeState := &annotateast.AnnotateTypeState{}
	for {
		oneType := parserOneType(l)
		typeState.ListType = append(typeState.ListType, oneType)

		if l.LookAheadKind() == annotatelexer.ATokenSepComma {
			l.NextTokenOfKind(annotatelexer.ATokenSepComma)
			// 是逗号， 表示是多个类型的之一
		} else {
			break
		}
	}

	// 获取这个state的多余注释
	typeState.Comment, typeState.CommentLoc = l.GetRemainComment()

	return typeState
}

// 解析@Alias
//---@alias NEW_NAME TYPE
func parserAliasState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// skip alias token
	l.NextTokenOfKind(annotatelexer.ATokenKwAlias)

	aliasState := &annotateast.AnnotateAliasState{}

	// 解析alias的名称
	aliasState.Name = l.NextIdentifier()
	aliasState.NameLoc = l.GetNowLoc()

	// 解析映射的具体type
	aliasState.AliasType = parserOneType(l)

	// 获取这个state的多余注释
	aliasState.Comment, aliasState.CommentLoc = l.GetRemainComment()

	return aliasState
}

// 解析@class
// ---@class MY_TYPE[:PARENT_TYPE] [@comment]
// ---@class MY_TYPE{:PARENT_TYPE [,PARENT_TYPE]}
func parserClassState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// skip class token
	l.NextTokenOfKind(annotatelexer.ATokenKwClass)

	classState := &annotateast.AnnotateClassState{}

	// 解析class的名称
	classState.Name = l.NextFieldName()
	classState.NameLoc = l.GetNowLoc()

	// 判断这个类是否有父类， 是否包含 :
	if l.LookAheadKind() == annotatelexer.ATokenSepColon {
		// 跳过冒号
		l.NextTokenOfKind(annotatelexer.ATokenSepColon)

		for {
			// 解析多个父类
			oneParentName := l.NextFieldName()
			classState.ParentNameList = append(classState.ParentNameList, oneParentName)
			classState.ParentLocList = append(classState.ParentLocList, l.GetNowLoc())

			if l.LookAheadKind() == annotatelexer.ATokenSepComma {
				// 是逗号， 表示有多个父类
				l.NextTokenOfKind(annotatelexer.ATokenSepComma)
			} else {
				break
			}
		}
	}

	// 获取这个state的多余注释
	classState.Comment, classState.CommentLoc = l.GetRemainComment()

	return classState
}

// 解析@overload
// ---@overload fun(list:table, sep:string):string
func parserOverloadState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// skip overload token
	l.NextTokenOfKind(annotatelexer.ATokenKwOverload)

	overloadState := &annotateast.AnnotateOverloadState{}

	// 解析后面的fun 函数
	subType := parserFunType(l)
	funcType, _ := subType.(*annotateast.FuncType)
	overloadState.OverFunType = funcType

	// 获取这个state的多余注释
	overloadState.Comment, overloadState.CommentLoc = l.GetRemainComment()

	return overloadState
}

// 解析@field
// ---@field [public|protected|private] field_name FIELDLTYPE[|OTHER_TYPE] [@comment]
func parserFieldState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// skip filed token
	l.NextTokenOfKind(annotatelexer.ATokenKwField)

	fieldState := &annotateast.AnnotateFieldState{}
	fieldState.FieldScopeType = annotateast.FieldScopePublic

	// 判断是否为public、protected、private 属性
	lookHeadKind := l.LookAheadKind()
	if lookHeadKind == annotatelexer.ATokenKwPubic ||
		lookHeadKind == annotatelexer.ATokenKwProtected ||
		lookHeadKind == annotatelexer.ATokenKwPrivate {

		if lookHeadKind == annotatelexer.ATokenKwProtected {
			fieldState.FieldScopeType = annotateast.FieldScopeProtected
		} else if lookHeadKind == annotatelexer.ATokenKwPrivate {
			fieldState.FieldScopeType = annotateast.FieldScopePrivate
		}
		l.NextToken()
	}

	// 获取name
	fieldState.Name = l.NextFieldName()
	fieldState.NameLoc = l.GetNowLoc()
	fieldState.FieldColonType = annotateast.FieldColonNo

	// 判断是否为 ：属性
	if l.LookAheadKind() == annotatelexer.ATokenSepColon {
		l.NextToken()
		fieldState.FieldColonType = annotateast.FieldColonYes
	}

	// 获取对应的type
	fieldState.FiledType = parserOneType(l)

	// 获取这个state的多余注释
	fieldState.Comment, fieldState.CommentLoc = l.GetRemainComment()

	return fieldState
}

// 解析@param
// ---@param param_name MY_TYPE[|other_type] [@comment]
func parserParamState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// skip param token
	l.NextTokenOfKind(annotatelexer.ATokenKwParam)

	paramState := &annotateast.AnnotateParamState{}

	// 获取参数的名字
	paramState.Name = l.NextParamName()
	paramState.NameLoc = l.GetNowLoc()

	// 判断是否为可选的 ？
	if l.LookAheadKind() == annotatelexer.ATokenOption {
		paramState.IsOptional = true
		l.NextToken()
	}

	// 获取参数的类型
	paramState.ParamType = parserOneType(l)

	// 获取这个state的多余注释
	paramState.Comment, paramState.CommentLoc = l.GetRemainComment()

	return paramState
}

// 解析@return
// ---@return RETURN_TYPE[|OTHER_TYPE] [@comment1]
// ---@return RETURN_TYPE1[|OTHER_TYPE], RETURN_TYPE2[|OTHER_TYPE] [@comment1] [@comment2]
func parserReturnState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// 前面的关键词为param 跳过
	l.NextTokenOfKind(annotatelexer.ATokenKwReturn)

	returnState := &annotateast.AnnotateReturnState{}

	// 循环获取多个返回类型
	for {
		oneType := parserOneType(l)
		returnState.ReturnTypeList = append(returnState.ReturnTypeList, oneType)

		if l.LookAheadKind() == annotatelexer.ATokenOption {
			returnState.ReturnOptionList = append(returnState.ReturnOptionList, true)
			l.NextToken()
		} else {
			returnState.ReturnOptionList = append(returnState.ReturnOptionList, false)
		}

		if l.LookAheadKind() == annotatelexer.ATokenSepComma {
			// 是逗号， 表示有多个返回值
			l.NextTokenOfKind(annotatelexer.ATokenSepComma)
		} else {
			break
		}
	}

	// 获取这个state的多余注释
	returnState.Comment, returnState.CommentLoc = l.GetRemainComment()

	return returnState
}

// 解析@generic
//---@generic T1 [: PARENT_TYPE] [, T2 [: PARENT_TYPE]] @comment @comment
func parserGenericState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// 前面的关键词为generic 跳过
	l.NextTokenOfKind(annotatelexer.ATokenKwGeneric)

	genericState := &annotateast.AnnotateGenericState{}

	// 循环处理这行定义的多个
	for {
		oneName := l.NextIdentifier()
		genericState.NameList = append(genericState.NameList, oneName)
		genericState.NameLocList = append(genericState.NameLocList, l.GetNowLoc())

		// 父的名称
		parentName := ""
		parentLoc := lexer.Location{}

		// 判断后面是否包含 :
		if l.LookAheadKind() == annotatelexer.ATokenSepColon {
			l.NextTokenOfKind(annotatelexer.ATokenSepColon)
			// 解析其父的名称
			parentName = l.NextIdentifier()
			parentLoc = l.GetNowLoc()
		}
		genericState.ParentNameList = append(genericState.ParentNameList, parentName)
		genericState.ParentLocList = append(genericState.ParentLocList, parentLoc)

		if l.LookAheadKind() == annotatelexer.ATokenSepComma {
			// 是逗号， 表示有多个generic
			l.NextTokenOfKind(annotatelexer.ATokenSepComma)
		} else {
			break
		}
	}

	// 获取这个state的多余注释
	genericState.Comment, genericState.CommentLoc = l.GetRemainComment()

	return genericState
}

// 解析@vararg
// ---@vararg TYPE
func parserVarargState(l *annotatelexer.AnnotateLexer) annotateast.AnnotateState {
	// 前面的关键词为vararg 跳过
	l.NextTokenOfKind(annotatelexer.ATokenKwVararg)

	varargState := &annotateast.AnnotateVarargState{}

	// 解析对应的类型
	varargState.VarargType = parserOneType(l)

	// 获取这个state的多余注释
	varargState.Comment, varargState.CommentLoc = l.GetRemainComment()

	return varargState
}
