package qunmaoniang

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/FloatTech/zbputils/ctxext"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

// 技能CD记录表
type cdsheet struct {
	Time    int64  // 时间
	GroupID int64  // 群号
	UserID  int64  // 用户
	ModeID  string // 技能类型
}

var sendtext = [...][]string{
	{ // 表白成功
		"是个勇敢的孩子(*/ω＼*) 今天的运气都降临在你的身边~\n\n",
		"(´･ω･`)对方答应了你 并表示愿意当你今天的专属猫娘\n\n",
	},
	{ // 表白失败
		"今天的运气有一点背哦~明天再试试叭",
		"_(:з」∠)_下次还有机会 咱抱抱你w",
		"今天失败了惹. 摸摸头~咱明天还有机会",
	},
	{ // ntr成功
		"因为你的个人魅力~~今天他就是你的了w\n\n",
	},
	{ // 离婚失败
		"丢弃失败啦~不要丢弃我嘛",
		"丢弃失败啦~不要丢弃我嘛",
	},
	{ // 离婚成功
		"丢弃成功力QWQ\n",
		"丢弃成功力QWQ\n",
	},
}

func init() {
	engine.OnRegex(`^设置CD为(\d+)小时`, zero.OnlyGroup, zero.AdminPermission, getdb).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			cdTime, err := strconv.ParseFloat(ctx.State["regex_matched"].([]string)[1], 64)
			if err != nil {
				ctx.SendChain(message.Text("[qunmaoniang]请设置纯数字\n", err))
				return
			}
			groupInfo, err := 民政局.查看设置(ctx.Event.GroupID)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			groupInfo.CDtime = cdTime
			err = 民政局.更新设置(groupInfo)
			if err != nil {
				ctx.SendChain(message.Text("[qunmaoniang]设置CD时长失败\n", err))
				return
			}
			ctx.SendChain(message.Text("设置成功"))
		})
	engine.OnRegex(`^(允许|禁止)(自由领养|牛头人)$`, zero.OnlyGroup, zero.AdminPermission, getdb).SetBlock(true).
		Handle(func(ctx *zero.Ctx) {
			status := ctx.State["regex_matched"].([]string)[1]
			mode := ctx.State["regex_matched"].([]string)[2]
			groupInfo, err := 民政局.查看设置(ctx.Event.GroupID)
			switch {
			case err != nil:
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			case mode == "自由领养":
				if status == "允许" {
					groupInfo.CanMatch = 1
				} else {
					groupInfo.CanMatch = 0
				}
			case mode == "牛头人":
				if status == "允许" {
					groupInfo.CanNtr = 1
				} else {
					groupInfo.CanNtr = 0
				}
			}
			err = 民政局.更新设置(groupInfo)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			ctx.SendChain(message.Text("设置成功"))
		})
	// 单身技能
	engine.OnRegex(`^(领养|找主人)\[CQ:at,qq=(\d+)\]`, zero.OnlyGroup, getdb, checkSingleDog).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			gid := ctx.Event.GroupID
			uid := ctx.Event.UserID
			choice := ctx.State["regex_matched"].([]string)[1]
			fiancee, _ := strconv.ParseInt(ctx.State["regex_matched"].([]string)[2], 10, 64)
			// 写入CD
			err := 民政局.记录CD(gid, uid, "领养")
			if err != nil {
				ctx.SendChain(message.At(uid), message.Text("[qunmaoniang]你的技能CD记录失败\n", err))
			}
			if uid == fiancee { // 如果是自己
				switch rand.Intn(3) {
				case 1:
					err := 民政局.登记(gid, uid, 0, "", "")
					if err != nil {
						ctx.SendChain(message.Text("[ERROR]:", err))
						return
					}
					ctx.SendChain(message.Text("你今天无法领猫娘哦~"))
				default:
					ctx.SendChain(message.Text("你今天无法领猫娘哦~"))
				}
				return
			}
			favor, err := 民政局.查好感度(uid, fiancee)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			if favor < 30 {
				favor = 30 // 保底30%概率
			}
			if rand.Intn(101) >= favor {
				ctx.SendChain(message.Text(sendtext[1][rand.Intn(len(sendtext[1]))]))
				return
			}
			// 去民政局登记
			var choicetext string
			switch choice {
			case "领养":
				err := 民政局.登记(gid, uid, fiancee, ctx.CardOrNickName(uid), ctx.CardOrNickName(fiancee))
				if err != nil {
					ctx.SendChain(message.Text("[ERROR]:", err))
					return
				}
				choicetext = "\n今天你的群猫娘是"
			default:
				err := 民政局.登记(gid, fiancee, uid, ctx.CardOrNickName(fiancee), ctx.CardOrNickName(uid))
				if err != nil {
					ctx.SendChain(message.Text("[ERROR]:", err))
					return
				}
				choicetext = "\n今天你的群主人是"
			}
			// 请大家吃席
			ctx.SendChain(
				message.Text(sendtext[0][rand.Intn(len(sendtext[0]))]),
				message.At(uid),
				message.Text(choicetext),
				message.Image("http://q4.qlogo.cn/g?b=qq&nk="+strconv.FormatInt(fiancee, 10)+"&s=640").Add("cache", 0),
				message.Text(
					"\n",
					"[", ctx.CardOrNickName(fiancee), "]",
					"(", fiancee, ")哒",
				),
			)
		})
	// NTR技能
	engine.OnRegex(`^当(\[CQ:at,qq=(\d+)\]\s?|(\d+))的仆人`, zero.OnlyGroup, getdb, checkMistress).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			gid := ctx.Event.GroupID
			uid := ctx.Event.UserID
			fid := ctx.State["regex_matched"].([]string)
			fiancee, _ := strconv.ParseInt(fid[2]+fid[3], 10, 64)
			// 写入CD
			err := 民政局.记录CD(gid, uid, "NTR")
			if err != nil {
				ctx.SendChain(message.At(uid), message.Text("[qunmaoniang]你的技能CD记录失败\n", err))
			}
			if fiancee == uid {
				ctx.SendChain(message.Text("今日获得成就：自我攻略"))
				return
			}
			favor, err := 民政局.查好感度(uid, fiancee)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			if favor < 30 {
				favor = 30 // 保底10%概率
			}
			if rand.Intn(101) >= favor/3 {
				ctx.SendChain(message.Text("失败了！可惜"))
				return
			}
			// 判断target是老公还是老婆
			var choicetext string
			var ntrID = uid
			var targetID = fiancee
			var greenID int64 // 被牛的
			fianceeInfo, err := 民政局.查户口(gid, fiancee)
			switch {
			case err != nil:
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			//case fianceeInfo.User == fiancee: // 是1
				//err = 民政局.离婚休妻(gid, fianceeInfo.Target)
				//if err != nil {
				//	ctx.SendChain(message.Text("ta不想和原来的对象分手...\n[error]", err))
				//	return
				//}
				//ntrID = fiancee
				//targetID = ctx.Event.UserID
				//greenID = fianceeInfo.Target
				//choicetext = "老公"
			//case fianceeInfo.Target == fiancee: // 是0
				//err = 民政局.离婚休夫(gid, fianceeInfo.User)
				//if err != nil {
				//	ctx.SendChain(message.Text("ta不想和原来的对象分手...\n[error]", err))
				//	return
				//}
				//greenID = fianceeInfo.Target
				//choicetext = "老婆"
			default:
				ctx.SendChain(message.Text("数据库发生问题力"))
				return
			}
			err = 民政局.登记(gid, ntrID, targetID, ctx.CardOrNickName(ntrID), ctx.CardOrNickName(targetID))
			if err != nil {
				ctx.SendChain(message.Text("[qunmaoniang]复婚登记失败力\n", err))
				return
			}
			favor, err = 民政局.更新好感度(uid, fiancee, -5)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
			}
			_, err = 民政局.更新好感度(uid, greenID, 5)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
			}
			// 输出结果
			ctx.SendChain(
				message.Text(sendtext[2][rand.Intn(len(sendtext[2]))]),
				message.At(uid),
				message.Text("今天你的群"+choicetext+"是"),
				message.Image("http://q4.qlogo.cn/g?b=qq&nk="+strconv.FormatInt(fiancee, 10)+"&s=640").Add("cache", 0),
				message.Text(
					"\n",
					"[", ctx.CardOrNickName(fiancee), "]",
					"(", fiancee, ")哒\n当前你们好感度为", favor,
				),
			)
		})
	// 做媒技能
	engine.OnRegex(`^做媒\s?\[CQ:at,qq=(\d+)\]\s?\[CQ:at,qq=(\d+)\]`, zero.OnlyGroup, zero.AdminPermission, getdb, checkMatchmaker).SetBlock(true).Limit(ctxext.LimitByUser).
		Handle(func(ctx *zero.Ctx) {
			gid := ctx.Event.GroupID
			uid := ctx.Event.UserID
			gayOne, _ := strconv.ParseInt(ctx.State["regex_matched"].([]string)[1], 10, 64)
			gayZero, _ := strconv.ParseInt(ctx.State["regex_matched"].([]string)[2], 10, 64)
			// 写入CD
			err := 民政局.记录CD(gid, uid, "做媒")
			if err != nil {
				ctx.SendChain(message.At(uid), message.Text("[qunmaoniang]你的技能CD记录失败\n", err))
			}
			favor, err := 民政局.查好感度(gayOne, gayZero)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			if favor < 30 {
				favor = 30 // 保底30%概率
			}
			if rand.Intn(101) >= favor {
				_, err = 民政局.更新好感度(uid, gayOne, -1)
				if err != nil {
					ctx.SendChain(message.Text("[ERROR]:", err))
				}
				_, err = 民政局.更新好感度(uid, gayZero, -1)
				if err != nil {
					ctx.SendChain(message.Text("[ERROR]:", err))
				}
				ctx.SendChain(message.Text(sendtext[1][rand.Intn(len(sendtext[1]))]))
				return
			}
			// 去民政局登记
			err = 民政局.登记(gid, gayOne, gayZero, ctx.CardOrNickName(gayOne), ctx.CardOrNickName(gayZero))
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			_, err = 民政局.更新好感度(uid, gayOne, 1)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
			}
			_, err = 民政局.更新好感度(uid, gayZero, 1)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
			}
			_, err = 民政局.更新好感度(gayOne, gayZero, 1)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
			}
			// 请大家吃席
			ctx.SendChain(
				message.At(uid),
				message.Text("恭喜你成功撮合了一对CP\n\n"),
				message.At(gayOne),
				message.Text("今天你的群老婆是"),
				message.Image("http://q4.qlogo.cn/g?b=qq&nk="+strconv.FormatInt(gayZero, 10)+"&s=640").Add("cache", 0),
				message.Text(
					"\n",
					"[", ctx.CardOrNickName(gayZero), "]",
					"(", gayZero, ")哒",
				),
			)
		})
	engine.OnFullMatchGroup([]string{"丢弃", "丢弃"}, zero.OnlyGroup, getdb, checkDivorce).Limit(ctxext.LimitByUser).SetBlock(true).
		Handle(func(ctx *zero.Ctx) {
			gid := ctx.Event.GroupID
			uid := ctx.Event.UserID
			// 写入CD
			err := 民政局.记录CD(gid, uid, "丢弃")
			if err != nil {
				ctx.SendChain(message.At(uid), message.Text("[qunmaoniang]你的技能CD记录失败\n", err))
			}
			mun := -1
			var fiancee int64
			userInfo, _ := 民政局.查户口(gid, uid)
			switch {
			case userInfo.User == uid:
				mun = 1
				fiancee = userInfo.Target
			case userInfo.Target == uid:
				mun = 0
				fiancee = userInfo.User
			}
			favor, err := 民政局.查好感度(uid, fiancee)
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			if favor < 20 {
				favor = 10
			}
			if rand.Intn(101) > 110-favor {
				ctx.SendChain(message.Text(sendtext[3][rand.Intn(len(sendtext[3]))]))
				return
			}
			switch mun {
			//case 1:
			//	err = 民政局.离婚休妻(gid, fiancee)
			//case 0:
			//	err = 民政局.离婚休夫(gid, fiancee)
			default:
				err = errors.New("用户数据查找发生错误")
			}
			if err != nil {
				ctx.SendChain(message.Text("[ERROR]:", err))
				return
			}
			ctx.SendChain(message.Text(sendtext[4][mun]))
		})
}

func (sql *婚姻登记) 判断CD(gid, uid int64, model string, cdtime float64) (ok bool, err error) {
	sql.Lock()
	defer sql.Unlock()
	// 创建群表格
	err = sql.db.Create("cdsheet", &cdsheet{})
	if err != nil {
		return false, err
	}
	limitID := "where GroupID is " + strconv.FormatInt(gid, 10) +
		" and UserID is " + strconv.FormatInt(uid, 10) +
		" and ModeID is '" + model + "'"
	if !sql.db.CanFind("cdsheet", limitID) {
		// 没有记录即不用比较
		return true, nil
	}
	cdinfo := cdsheet{}
	_ = sql.db.Find("cdsheet", &cdinfo, limitID)
	if time.Since(time.Unix(cdinfo.Time, 0)).Hours() > cdtime {
		// 如果CD已过就删除
		err = sql.db.Del("cdsheet", limitID)
		return true, err
	}
	return false, nil
}

func (sql *婚姻登记) 记录CD(gid, uid int64, mode string) error {
	sql.Lock()
	defer sql.Unlock()
	return sql.db.Insert("cdsheet", &cdsheet{
		Time:    time.Now().Unix(),
		GroupID: gid,
		UserID:  uid,
		ModeID:  mode,
	})
}

//func (sql *婚姻登记) 离婚休妻(gid, wife int64) error {
	//sql.Lock()
	//defer sql.Unlock()
	//gidstr := "group" + strconv.FormatInt(gid, 10)
	//wifestr := strconv.FormatInt(wife, 10)
	//return sql.db.Del(gidstr, "where target = "+wifestr)
//}

//func (sql *婚姻登记) 离婚休夫(gid, husband int64) error {
	//sql.Lock()
	//defer sql.Unlock()
	//gidstr := "group" + strconv.FormatInt(gid, 10)
	//husbandstr := strconv.FormatInt(husband, 10)
	//return sql.db.Del(gidstr, "where user = "+husbandstr)
//}

// 注入判断 是否单身条件
func checkSingleDog(ctx *zero.Ctx) bool {
	gid := ctx.Event.GroupID
	uid := ctx.Event.UserID
	fiancee, err := strconv.ParseInt(ctx.State["regex_matched"].([]string)[2], 10, 64)
	if err != nil {
		ctx.SendChain(message.Text("额,你的target好像不存在?"))
		return false
	}
	// 判断是否需要重置
	err = 民政局.开门时间(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	// 判断是否符合条件
	groupInfo, err := 民政局.查看设置(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	if groupInfo.CanMatch == 0 {
		ctx.SendChain(message.Text("你群包分配,别在娶妻上面下功夫，好好水群"))
		return false
	}
	// 判断CD
	ok, err := 民政局.判断CD(gid, uid, "领养", groupInfo.CDtime)
	switch {
	case err != nil:
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	case !ok:
		ctx.SendChain(message.Text("你的技能还在CD中..."))
		return false
	}
	// 获取用户信息
	userInfo, _ := 民政局.查户口(gid, uid)
	switch {
	case userInfo != (userinfo{}) && (userInfo.Target == 0 || userInfo.User == 0): // 如果是单身贵族
		ctx.SendChain(message.Text("今天的你无法领养猫娘哦"))
		return false
	case userInfo.Target == fiancee || userInfo.User == fiancee:
		ctx.SendChain(message.Text("笨蛋！ta已经是你的猫娘啦"))
		return false
	case userInfo.User == uid: // 如果如为攻
		ctx.SendChain(message.Text("失败"))
		return false
	case userInfo.Target == uid: // 如果为受
		ctx.SendChain(message.Text("失败"))
		return false
	}
	fianceeInfo, _ := 民政局.查户口(gid, fiancee)
	switch {
	case fianceeInfo != (userinfo{}) && (fianceeInfo.Target == 0 || fianceeInfo.User == 0): // 如果是单身贵族
		ctx.SendChain(message.Text("今天的你无法领养猫娘哦"))
		return false
	case fianceeInfo.User == fiancee: // 如果如为攻
		ctx.SendChain(message.Text("他有猫娘了哦"))
		return false
	case fianceeInfo.Target == fiancee: // 如果为受
		ctx.SendChain(message.Text("ta被别人领养领"))
		return false
	}
	return true
}

// 注入判断 是否满足小三要求
func checkMistress(ctx *zero.Ctx) bool {
	gid := ctx.Event.GroupID
	uid := ctx.Event.UserID
	fiancee, err := strconv.ParseInt(ctx.State["regex_matched"].([]string)[2], 10, 64)
	if err != nil {
		ctx.SendChain(message.Text("额,你的target好像不存在?"))
		return false
	}
	// 判断是否需要重置
	err = 民政局.开门时间(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	// 判断是否符合条件
	groupInfo, err := 民政局.查看设置(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	if groupInfo.CanNtr == 0 {
		ctx.SendChain(message.Text("你群发布了领养禁止令，放弃吧"))
		return false
	}
	// 判断CD
	ok, err := 民政局.判断CD(gid, uid, "领养", groupInfo.CDtime)
	switch {
	case err != nil:
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	case !ok:
		ctx.SendChain(message.Text("你的技能还在CD中..."))
		return false
	}
	// 获取用户信息
	fianceeInfo, _ := 民政局.查户口(gid, fiancee)
	switch {
	case fianceeInfo == (userinfo{}): // 如果是空数据
		ctx.SendChain(message.Text("ta还没有主人哦"))
		return false
	case fianceeInfo.Target == 0 || fianceeInfo.User == 0: // 如果是单身贵族
		ctx.SendChain(message.Text("今天的你无法领养猫娘哦"))
		return false
	case fianceeInfo.Target == uid || fianceeInfo.User == uid:
		ctx.SendChain(message.Text("笨蛋！ta已经是你的猫娘啦"))
		return false
	}
	// 获取用户信息
	userInfo, _ := 民政局.查户口(gid, uid)
	switch {
	case userInfo != (userinfo{}) && (userInfo.Target == 0 || userInfo.User == 0): // 如果是单身贵族
		ctx.SendChain(message.Text("今天的你无法领养猫娘哦"))
		return false
	case userInfo.User == uid: // 如果如为攻
		ctx.SendChain(message.Text("今天的你无法领养猫娘哦"))
		return false
	case userInfo.Target == uid: // 如果为受
		ctx.SendChain(message.Text("今天的你无法领养猫娘哦"))
		return false
	}
	return true
}

func checkDivorce(ctx *zero.Ctx) bool {
	gid := ctx.Event.GroupID
	uid := ctx.Event.UserID
	// 判断是否需要重置
	err := 民政局.开门时间(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	// 判断是否符合条件
	userInfo, _ := 民政局.查户口(gid, uid)
	if userInfo == (userinfo{}) { // 如果空数据
		ctx.SendChain(message.Text("今天你还没领养/被领养哦"))
		return false
	}
	// 获取CD
	groupInfo, err := 民政局.查看设置(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	ok, err := 民政局.判断CD(gid, uid, "丢弃", groupInfo.CDtime)
	switch {
	case err != nil:
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	case !ok:
		ctx.SendChain(message.Text("你的技能还在CD中..."))
		return false
	}
	return true
}

func checkMatchmaker(ctx *zero.Ctx) bool {
	gid := ctx.Event.GroupID
	uid := ctx.Event.UserID
	gayOne, err := strconv.ParseInt(ctx.State["regex_matched"].([]string)[1], 10, 64)
	if err != nil {
		ctx.SendChain(message.Text("失败"))
		return false
	}
	gayZero, err := strconv.ParseInt(ctx.State["regex_matched"].([]string)[2], 10, 64)
	if err != nil {
		ctx.SendChain(message.Text("失败"))
		return false
	}
	if gayOne == uid || gayZero == uid {
		ctx.SendChain(message.Text("失败"))
		return false
	}
	if gayOne == gayZero {
		ctx.SendChain(message.Text("失败"))
		return false
	}
	// 判断是否需要重置
	err = 民政局.开门时间(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	// 获取CD
	groupInfo, err := 民政局.查看设置(gid)
	if err != nil {
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	}
	ok, err := 民政局.判断CD(gid, uid, "none", groupInfo.CDtime)
	switch {
	case err != nil:
		ctx.SendChain(message.Text("[ERROR]:", err))
		return false
	case !ok:
		ctx.SendChain(message.Text("你的技能还在CD中..."))
		return false
	}
	gayOneInfo, _ := 民政局.查户口(gid, gayOne)
	switch {
	case gayOneInfo != (userinfo{}) && (gayOneInfo.Target == 0 || gayOneInfo.User == 0): // 如果是单身贵族
		ctx.SendChain(message.Text("失败"))
		return false
	case gayOneInfo.Target == gayZero || gayOneInfo.User == gayZero:
		ctx.SendChain(message.Text("失败"))
		return false
	case gayOneInfo != (userinfo{}): // 如果不是单身
		ctx.SendChain(message.Text("失败"))
		return false
	}
	// 获取用户信息
	gayZeroInfo, _ := 民政局.查户口(gid, gayZero)
	switch {
	case gayOneInfo != (userinfo{}) && (gayZeroInfo.Target == 0 || gayZeroInfo.User == 0): // 如果是单身贵族
		ctx.SendChain(message.Text("失败"))
		return false
	case gayZeroInfo != (userinfo{}): // 如果不是单身
		ctx.SendChain(message.Text("失败"))
		return false
	}
	return true
}
