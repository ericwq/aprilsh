/*

MIT License

Copyright (c) 2022 wangqi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package network

import (
	"strings"
	"testing"
)

func TestCompressor(t *testing.T) {
	tc := []struct {
		name string
		text string
	}{
		{
			"compress english text",
			`A string is a sequence of one or more characters (letters, numbers, symbols) that can be either a constant or a variable. Made up of Unicode, strings are immutable sequences, meaning they are unchanging.
Because text is such a common form of data that we use in everyday life, the string data type is a very important building block of programming.
This Go tutorial will go over how to create and print strings, how to concatenate and replicate strings, and how to store strings in variables.`,
		},
		{
			"compress chinese text",
			`大江東去，浪淘盡，千古風流人物。
故壘西邊，人道是，三國周郎赤壁。
亂石穿空，驚濤拍岸，捲起千堆雪。
江山如畫，一時多少豪傑。
遙想公瑾當年，小喬初嫁了，雄姿英發。
羽扇綸巾，談笑間，檣櫓灰飛煙滅。(檣櫓 一作：強擄)
故國神遊，多情應笑我，早生華髮。
人生如夢，一尊還酹江月。`,
		},
	}

	cp := GetCompressor()
	for _, v := range tc {
		mid, err := cp.Compress([]byte(v.text))
		if err != nil {
			t.Errorf("%q compress fail, %s\n", v.name, err)
		}

		// fmt.Printf("text=%d mid=%d\n", len(v.text), len(mid))
		text, err := cp.Uncompress(mid)
		if err != nil {
			t.Errorf("%q uncompress fail, %s\n", v.name, err)
		}

		got := string(text)
		if got != v.text {
			t.Errorf("%q expect \n%q, got\n%q\n", v.name, v.text, got)
		}
	}
}

func TestCompressorOversize(t *testing.T) {
	// 长恨歌 - 唐代·白居易
	text := `汉皇重色思倾国，御宇多年求不得。 杨家有女初长成，养在深闺人未识。 天生丽质难自弃，一朝选在君王侧。 回眸一笑百媚生，六宫粉黛无颜色。 春寒赐浴华清池，温泉水滑洗凝脂。 侍儿扶起娇无力，始是新承恩泽时。 云鬓花颜金步摇，芙蓉帐暖度春宵。 春宵苦短日高起，从此君王不早朝。 承欢侍宴无闲暇，春从春游夜专夜。 后宫佳丽三千人，三千宠爱在一身。 金屋妆成娇侍夜，玉楼宴罢醉和春。 姊妹弟兄皆列土，可怜光彩生门户。 遂令天下父母心，不重生男重生女。 骊宫高处入青云，仙乐风飘处处闻。 缓歌慢舞凝丝竹，尽日君王看不足。 渔阳鼙鼓动地来，惊破霓裳羽衣曲。 九重城阙烟尘生，千乘万骑西南行。 翠华摇摇行复止，西出都门百余里。 六军不发无奈何，宛转蛾眉马前死。 花钿委地无人收，翠翘金雀玉搔头。 君王掩面救不得，回看血泪相和流。 黄埃散漫风萧索，云栈萦纡登剑阁。 峨嵋山下少人行，旌旗无光日色薄。 蜀江水碧蜀山青，圣主朝朝暮暮情。 行宫见月伤心色，夜雨闻铃肠断声。 天旋地转回龙驭，到此踌躇不能去。 马嵬坡下泥土中，不见玉颜空死处。 君臣相顾尽沾衣，东望都门信马归。 归来池苑皆依旧，太液芙蓉未央柳。 芙蓉如面柳如眉，对此如何不泪垂。 春风桃李花开日，秋雨梧桐叶落时。 西宫南内多秋草，落叶满阶红不扫。(花开日 一作：花开夜；南内 一作：南苑) 梨园弟子白发新，椒房阿监青娥老。 夕殿萤飞思悄然，孤灯挑尽未成眠。 迟迟钟鼓初长夜，耿耿星河欲曙天。 鸳鸯瓦冷霜华重，翡翠衾寒谁与共。 悠悠生死别经年，魂魄不曾来入梦。 临邛道士鸿都客，能以精诚致魂魄。 为感君王辗转思，遂教方士殷勤觅。 排空驭气奔如电，升天入地求之遍。 上穷碧落下黄泉，两处茫茫皆不见。 忽闻海上有仙山，山在虚无缥渺间。 楼阁玲珑五云起，其中绰约多仙子。 中有一人字太真，雪肤花貌参差是。 金阙西厢叩玉扃，转教小玉报双成。 闻道汉家天子使，九华帐里梦魂惊。 揽衣推枕起徘徊，珠箔银屏迤逦开。 云鬓半偏新睡觉，花冠不整下堂来。 风吹仙袂飘飖举，犹似霓裳羽衣舞。 玉容寂寞泪阑干，梨花一枝春带雨。(阑 通：栏；飘飘 一作：飘飖) 含情凝睇谢君王，一别音容两渺茫。 昭阳殿里恩爱绝，蓬莱宫中日月长。 回头下望人寰处，不见长安见尘雾。 惟将旧物表深情，钿合金钗寄将去。 钗留一股合一扇，钗擘黄金合分钿。 但教心似金钿坚，天上人间会相见。 临别殷勤重寄词，词中有誓两心知。 七月七日长生殿，夜半无人私语时。 在天愿作比翼鸟，在地愿为连理枝。 天长地久有时尽，此恨绵绵无绝期。`

	// fmt.Printf("len=%d\n", len(text)) output:3034
	cp := GetCompressor()
	mid, _ := cp.Compress([]byte(text))
	// fmt.Printf("len=%d\n", len(mid)) output:1974
	_, err := cp.Uncompress(mid)

	expectStr := "content length exceed the buffer size"
	if !strings.Contains(err.Error(), expectStr) {
		t.Errorf("compressor over size expect %q, got %q\n", expectStr, err)
	}
}

func TestCompressorFail(t *testing.T) {
	text := "long long ago"
	cp := GetCompressor()
	mid, _ := cp.Compress([]byte(text))

	// fmt.Printf("mid=% x\n", mid)
	mid[4] = 78

	_, err := cp.Uncompress(mid)
	expectStr := "invalid checksum"
	if !strings.Contains(err.Error(), expectStr) {
		t.Errorf("compressor fail expect %q, got %q\n", expectStr, err)
	}
}
