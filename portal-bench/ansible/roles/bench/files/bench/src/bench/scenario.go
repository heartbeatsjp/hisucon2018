package bench

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	loginReg = regexp.MustCompile(`^/login$`)
)

func checkHTML(f func(*http.Response, *goquery.Document) error) func(*http.Response, *bytes.Buffer) error {
	return func(res *http.Response, body *bytes.Buffer) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return fatalErrorf("ページのHTMLがパースできませんでした")
		}
		return f(res, doc)
	}
}

func genPostImageBody(fileName string, title string, text string) (*bytes.Buffer, string, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("title", title)
	writer.WriteField("text", text)

	imageNum := rand.Perm(len(UploadFileImages) - 1)[0]
	image := UploadFileImages[imageNum]

	uploadFile := filepath.Join(DataPath, image.Path)
	fileWriter, err := writer.CreateFormFile("upload", fileName)
	if err != nil {
		return nil, "", err
	}

	readFile, err := os.Open(uploadFile)
	if err != nil {
		return nil, "", err
	}
	defer readFile.Close()

	io.Copy(fileWriter, readFile)
	writer.Close()

	return body, writer.FormDataContentType(), err
}

func checkRedirectStatusCode(res *http.Response, body *bytes.Buffer) error {
	if res.StatusCode == 302 || res.StatusCode == 303 {
		return nil
	}
	return fmt.Errorf("期待していないステータスコード %d Expected 302 or 303", res.StatusCode)
}

func PreAddUser(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	_, checker2, push2 := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()
	defer push2()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     "suzuki",
			"password": "suzuki201808",
		},
		Description: "管理者ログインできること",
	})
	if err != nil {
		return err
	}

	newUser := RandomAlphabetString(16)
	newPass := RandomAlphabetString(16)

	rand.Seed(time.Now().UnixNano())
	is_admin := strconv.Itoa(rand.Intn(2))
	err = checker.Play(ctx, &CheckAction{
		Method: "POST",
		Path:   "/user/add/",
		PostData: map[string]string{
			"username": newUser,
			"password": newPass,
			"is_admin": is_admin,
		},
		Description: "新規ユーザが作成できること",
	})
	if err != nil {
		return err
	}

	err = checker2.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     newUser,
			"password": newPass,
		},
		Description: "作成したユーザでログインできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err

	}
	err = checker2.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	addUser := &AppUser{
		Name:     newUser,
		Password: newPass,
	}

	state.userMap[addUser.Name] = addUser
	state.users = append(state.users, addUser)

	return nil
}

// ログインユーザが投稿した記事、コメントは編集・削除ボタンが表示される
// ログインユーザ以外の記事、コメントに対しては編集・削除ボタンが表示されない
func CheckLayoutPreTest(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     "suzuki",
			"password": "suzuki201808",
		},
		Description: "存在するユーザでログインできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/",
		ExpectedStatusCode: 200,
		Description:        "ユーザ情報ページが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if doc.Find("body > div.container > p").Text() != "投稿がありません" {
				return fatalErrorf("投稿部分のフォーマットに誤りがあります。")
			}

			return nil
		}),
	})
	if err != nil {
		return err
	}

	title := RandomAlphabetString(16)
	text := RandomAlphabetString(256)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/edudaily/new/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿",
		PostData: map[string]string{
			"title": title,
			"text":  text,
		},
	})
	if err != nil {
		return err
	}

	textURL := ""
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/",
		ExpectedStatusCode: 200,
		Description:        "個人ユーザ情報が表示できること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > div .jumbotron > h2").First().Text() != title {
				return fatalErrorf("件名が正常に登録されてません。")
			}
			textURL, _ = doc.Find("body > div .jumbotron > p.lead > a").First().Attr("href")

			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "新規投稿したドキュメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if !strings.Contains(doc.Find("body > div .jumbotron > p").Text(), text) {
				return fatalErrorf("登録した本文が正常に表示されていません")
			}
			if doc.Find("body > div.container > div.jumbotron > div.btn-toolbar.float-right > form > button.btn.btn-success").Text() != "コメント" {
				return fatalErrorf("コメントボタンが適切に表示されていません。")
			}
			if doc.Find("body > div.container > div.jumbotron > div.btn-toolbar.float-right > form > button.btn.btn-primary").Text() != "編集" {
				return fatalErrorf("編集ボタンが適切に表示されていません。")
			}
			if doc.Find("body > div.container > div.jumbotron > div.btn-toolbar.float-right > form > button.btn.btn-danger").Text() != "削除" {
				return fatalErrorf("削除ボタンが適切に表示されていません。")
			}

			return nil
		}),
	})
	if err != nil {
		return err
	}

	comText1 := RandomAlphabetString(10)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿へのコメントを追加1",
		PostData: map[string]string{
			"text": comText1,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "新規投稿したドキュメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if !strings.Contains(doc.Find("body > div.container > div.card > div.card-body > p").Text(), comText1) {
				return fatalErrorf("登録したコメントが正常に表示されていません")
			}
			if doc.Find("body > div > div.card > div.card-body > div.btn-toolbar.float-right > form > button.btn.btn-primary").Text() != "編集" {
				return fatalErrorf("編集ボタンが適切に表示されていません。")
			}
			if doc.Find("body > div > div.card > div.card-body > div.btn-toolbar.float-right > form > button.btn.btn-danger").Text() != "削除" {
				return fatalErrorf("削除ボタンが適切に表示されていません。")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     "sato",
			"password": "sato201808",
		},
		Description: "存在するユーザでログインできること",
	})
	if err != nil {
		return err
	}
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "他人の投稿した記事が表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if !strings.Contains(doc.Find("body > div .jumbotron > p").Text(), text) {
				return fatalErrorf("登録した本文が正常に表示されていません")
			}
			if doc.Find("body > div.container > div.jumbotron > div.btn-toolbar.float-right > form > button.btn.btn-success").Text() != "コメント" {
				return fatalErrorf("コメントボタンが適切に表示されていません。")
			}
			if doc.Find("body > div.container > div.jumbotron > div.btn-toolbar.float-right > form > button.btn.btn-primary").Size() != 0 {
				return fatalErrorf("対象ユーザ以外で編集ボタンが表示されています。")
			}
			if doc.Find("body > div.container > div.jumbotron > div.btn-toolbar.float-right > form > button.btn.btn-danger").Size() != 0 {
				return fatalErrorf("対象ユーザ以外で削除ボタンが表示されています。")
			}

			return nil
		}),
	})
	if err != nil {
		return err
	}

	comText2 := RandomAlphabetString(10)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "他人投稿へのコメントを追加1",
		PostData: map[string]string{
			"text": comText2,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "新規投稿したドキュメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if !strings.Contains(doc.Find("body > div.container > div.card > div.card-body > p").Text(), comText2) {
				return fatalErrorf("登録したコメントが正常に表示されていません")
			}
			if doc.Find("body > div > div.card > div.card-body > div.btn-toolbar.float-right > form > button.btn.btn-primary").Size() != 1 {
				return fatalErrorf("対象ユーザ以外で編集ボタンが表示されています。")
			}
			if doc.Find("body > div > div.card > div.card-body > div.btn-toolbar.float-right > form > button.btn.btn-danger").Size() != 1 {
				return fatalErrorf("対象ユーザ以外で削除ボタンが表示されています。")
			}

			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil
}

// 投稿記事は更新日の降順であることを確認
// コメントは作成日の降順であることを確認
func CheckOrder(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
		Description: "存在するユーザでログインできること",
	})
	if err != nil {
		return err
	}

	title1 := RandomAlphabetString(16)
	text1 := RandomAlphabetString(256)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/edudaily/new/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿",
		PostData: map[string]string{
			"title": title1,
			"text":  text1,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/",
		ExpectedStatusCode: 200,
		Description:        "投稿が表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if doc.Find("body > div.container > div.jumbotron > h2.display-4").First().Text() != title1 {
				return fatalErrorf("記事が投稿されていません。")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	title2 := RandomAlphabetString(16)
	text2 := RandomAlphabetString(256)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/edudaily/new/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿",
		PostData: map[string]string{
			"title": title2,
			"text":  text2,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/",
		ExpectedStatusCode: 200,
		Description:        "投稿が表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if !strings.Contains(doc.Find("body > div.container > div.jumbotron > h2.display-4").Text(), title2) {
				return fatalErrorf("投稿したタイトルが正常に表示されていません")
			}
			if doc.Find("body > div.container > div.jumbotron > h2.display-4").First().Text() != title2 {
				return fatalErrorf("記事の表示順序が違います。")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	textURL := ""
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/",
		ExpectedStatusCode: 200,
		Description:        "個人ユーザ情報が表示できること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			textURL, _ = doc.Find("body > div .jumbotron > p.lead > a").First().Attr("href")

			return nil
		}),
	})
	if err != nil {
		return err
	}

	comText1 := RandomAlphabetString(10)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿へのコメントを追加1",
		PostData: map[string]string{
			"text": comText1,
		},
	})
	if err != nil {
		return err
	}

	comText2 := RandomAlphabetString(20)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿へのコメントを追加2",
		PostData: map[string]string{
			"text": comText2,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "新規投稿したコメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			comment := doc.Find("body > div.container > div.card > div.card-body > p").Text()
			comText1location := strings.Index(comment, comText1)
			comText2location := strings.Index(comment, comText2)
			if comText1location < comText2location {
				return fatalErrorf("記事の順番が違います")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil

}

// - "/" へのアクセスは "/login" へ遷移されることを確認
// - "/login" ページの表示されることを確認
// -   - プロジェクト名が「ハートビーツ研修ブログ」であることを確認
// -   - name 入力欄があることを確認
// -   - password 入力欄があることを確認
// -   - パスワード保存チェックボックスがあることを確認
// -   - ログインボタンがあることを確認
// - "/edudaily/8" へのアクセスは "/login" へ遷移されることを確認
// - "/user/" へのアクセスは "/login" へ遷移されることを確認
// - "/edudaily/new/" へのアクセスは "/login" へ遷移されることを確認
// - "/edudaily/2/new_com/?" へのアクセスは "/login" へ遷移されることを確認
func CheckNotLoggedInUser(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	checker.ResetCookie()

	err := checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "リダイレクトされること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/login",
		ExpectedStatusCode: 200,
		Description:        "ページが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if doc.Find("#name").Size() != 1 {
				return fatalErrorf("入力フォーム適切に表示されていません")
			}
			if doc.Find("#password").Size() != 1 {
				return fatalErrorf("入力フォーム適切に表示されていません")
			}
			if doc.Find("#remember_me").Size() != 1 {
				return fatalErrorf("入力フォーム適切に表示されていません")
			}
			loginText, _ := doc.Find("#submit").Attr("value")
			if loginText != "ログイン" {
				return fatalErrorf("ログインボタンが適切に表示されていません")
			}

			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:           "GET",
		Path:             "/edudaily/8",
		CheckFunc:        checkRedirectStatusCode,
		ExpectedLocation: loginReg,
		Description:      "ログインページにリダイレクトされること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:           "GET",
		Path:             "/user/",
		CheckFunc:        checkRedirectStatusCode,
		ExpectedLocation: loginReg,
		Description:      "ログインページにリダイレクトされること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:           "GET",
		Path:             "/edudaily/new/",
		CheckFunc:        checkRedirectStatusCode,
		ExpectedLocation: loginReg,
		Description:      "ログインページにリダイレクトされること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:           "GET",
		Path:             fmt.Sprintf("/edudaily/%d/new_com/?", 2),
		CheckFunc:        checkRedirectStatusCode,
		ExpectedLocation: loginReg,
		Description:      "ログインページにリダイレクトされること",
	})
	if err != nil {
		return err
	}

	return nil
}

// - 下記ファイルへアクセスできることを確認
//     - "/bootstrap/static/css/bootstrap.min.css?bootstrap=4.0.0"
//     - "/bootstrap/static/css/fontawesome-all.min.css?bootstrap=4.0.0"
//     - "/bootstrap/static/jquery.min.js?bootstrap=4.0.0"
//     - "/bootstrap/static/umd/popper.min.js?bootstrap=4.0.0"
//     - "/bootstrap/static/js/bootstrap.min.js?bootstrap=4.0.0"
// - 画像ファイルをアップロードすることができる
func CheckStaticFiles(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	for _, staticFile := range StaticFiles {
		sf := staticFile
		err := checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               sf.Path,
			ExpectedStatusCode: 200,
			Description:        "静的ファイルが取得できること",
			CheckFunc: func(res *http.Response, body *bytes.Buffer) error {
				hasher := md5.New()
				_, err := io.Copy(hasher, body)
				if err != nil {
					return fatalErrorf("レスポンスボディの取得に失敗 %v", err)
				}
				return nil
			},
		})
		if err != nil {
			return err
		}
	}

	imageNum := rand.Perm(len(StaticFileImages) - 1)[0]
	image := StaticFileImages[imageNum]
	err := checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               image.Path,
		ExpectedStatusCode: 200,
		Description:        "静的ファイルが取得できること",
		CheckFunc: func(res *http.Response, body *bytes.Buffer) error {
			hasher := md5.New()
			_, err := io.Copy(hasher, body)
			if err != nil {
				return fatalErrorf("レスポンスボディの取得に失敗 %v", err)
			}
			return nil
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// - 存在するユーザでログインすることを確認
// - ログアウトできることを確認
// - 存在しないユーザではログインできないことを確認
func CheckLogin(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
		Description: "存在するユーザでログインできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "POST",
		Path:               "/login",
		ExpectedStatusCode: 403,
		PostData: map[string]string{
			"name":     RandomAlphabetString(30),
			"password": RandomAlphabetString(30),
		},
		Description: "存在しないユーザでログインできないこと",
	})
	if err != nil {
		return err
	}

	return nil
}

func CheckAddUser(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     "suzuki",
			"password": "suzuki201808",
		},
		Description: "管理者ログインできること",
	})
	if err != nil {
		return err
	}

	newUser := RandomAlphabetString(16)
	newPass := RandomAlphabetString(16)
	err = checker.Play(ctx, &CheckAction{
		Method:             "POST",
		Path:               "/login",
		ExpectedStatusCode: 403,
		PostData: map[string]string{
			"name":     newUser,
			"password": newPass,
		},
		Description: "登録前のユーザでログインできないこと",
	})
	if err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())
	is_admin := strconv.Itoa(rand.Intn(2))
	err = checker.Play(ctx, &CheckAction{
		Method: "POST",
		Path:   "/user/add/",
		PostData: map[string]string{
			"username": newUser,
			"password": newPass,
			"is_admin": is_admin,
		},
		Description: "新規ユーザが作成できること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "POST",
		Path:               "/user/add/",
		ExpectedStatusCode: 409,
		PostData: map[string]string{
			"username": newUser,
			"password": newPass,
		},
		Description: "登録済のユーザ名が使えないこと",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     newUser,
			"password": newPass,
		},
		Description: "作成したユーザでログインできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil
}

func CheckLayout(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
		Description: "存在するユーザでログインできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/edudaily/new/",
		ExpectedStatusCode: 200,
		Description:        "新規投稿ページが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if doc.Find("#title").Size() != 1 {
				return fatalErrorf("タイトルフォームが適切に表示されていません")
			}
			if doc.Find("#text").Size() != 1 {
				return fatalErrorf("本文フォームが適切に表示されていません")
			}
			if doc.Find("#upload").Size() != 1 {
				return fatalErrorf("画像フォームが適切に表示されていません")
			}
			if doc.Find("#make").Size() != 1 {
				return fatalErrorf("作成フォームが適切に表示されていません")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/edit/",
		ExpectedStatusCode: 200,
		Description:        "ユーザ編集ページが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if doc.Find("#username").Size() != 1 {
				return fatalErrorf("ユーザ名が適切に表示されていません")
			}
			if doc.Find("#password").Size() != 1 {
				return fatalErrorf("パスワードフォームが適切に表示されていません")
			}
			/*
				if user.IsAdmin == "1" {
					if doc.Find("#is_admin").Size() != 1 {
						return fatalErrorf("管理者権限フォームが適切に表示されていません")
					}
				} else {
					if doc.Find("#is_admin").Size() != 0 {
						return fatalErrorf("管理者権限フォームが不正に表示されています。")
					}
				}
			*/
			if doc.Find("#icon").Size() != 1 {
				return fatalErrorf("アイコンフォームが適切に表示されていません")
			}
			if doc.Find("#submit").Size() != 1 {
				return fatalErrorf("送信フォームが適切に表示されていません")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/edudaily/77/new_com/?",
		ExpectedStatusCode: 200,
		Description:        "コメント追加ページが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
				return fatalErrorf("プロジェクト名が適切に表示されていません")
			}
			if doc.Find("#text").Size() != 1 {
				return fatalErrorf("本文フォーム名が適切に表示されていません")
			}
			if doc.Find("#submit").Size() != 1 {
				return fatalErrorf("送信フォームが適切に表示されていません")
			}
			return nil
		}),
	})

	if err != nil {
		return err
	}

	if user.IsAdmin == "1" {
		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               "/user/add/",
			ExpectedStatusCode: 200,
			Description:        "ユーザ追加ページが表示されること",
			CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
				if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
					return fatalErrorf("プロジェクト名が適切に表示されていません")
				}
				if doc.Find("#username").Size() != 1 {
					return fatalErrorf("ユーザ名が適切に表示されていません")
				}
				if doc.Find("#password").Size() != 1 {
					return fatalErrorf("パスワードフォームが適切に表示されていません")
				}
				if doc.Find("#is_admin").Size() != 1 {
					return fatalErrorf("管理者権限フォームが適切に表示されていません")
				}
				if doc.Find("#icon").Size() != 1 {
					return fatalErrorf("アイコンフォームが適切に表示されていません")
				}
				if doc.Find("#submit").Size() != 1 {
					return fatalErrorf("送信フォームが適切に表示されていません")
				}
				return nil
			}),
		})
		if err != nil {
			return err
		}

		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               "/user/admin/edit/1",
			ExpectedStatusCode: 200,
			Description:        "ユーザ編集ページが表示されること",
			CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
				if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
					return fatalErrorf("プロジェクト名が適切に表示されていません")
				}
				if doc.Find("#username").Size() != 1 {
					return fatalErrorf("ユーザ名が適切に表示されていません")
				}
				if doc.Find("#password").Size() != 1 {
					return fatalErrorf("パスワードフォームが適切に表示されていません")
				}
				if doc.Find("#is_admin").Size() != 1 {
					return fatalErrorf("管理者権限フォームが適切に表示されていません")
				}
				if doc.Find("#icon").Size() != 1 {
					return fatalErrorf("アイコンフォームが適切に表示されていません")
				}
				if doc.Find("#submit").Size() != 1 {
					return fatalErrorf("送信フォームが適切に表示されていません")
				}
				return nil
			}),
		})
		if err != nil {
			return err
		}

		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               "/user/all/",
			ExpectedStatusCode: 200,
			Description:        "ユーザ一覧ページが表示されること",
			CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
				if doc.Find("body > nav > a").Text() != "ハートビーツ研修共有ブログ" {
					return fatalErrorf("プロジェクト名が適切に表示されていません")
				}
				if doc.Find("body > div.container > table.table > tbody > tr > td > p > a.btn.btn-primary").First().Text() != "編集" {
					return fatalErrorf("編集ボタンが適切に表示されていません。")
				}
				if doc.Find("body > div.container > table.table > tbody > tr > td > form > button.btn.btn-danger").First().Text() != "削除" {
					return fatalErrorf("削除ボタンが適切に表示されていません。")
				}
				if doc.Find("body > div.container > table.table > tbody > tr > td > form > button.btn.btn-info").First().Text() != "戻す" {
					return fatalErrorf("戻すボタンが適切に表示されていません。")
				}
				if doc.Find("body > div.container > a.btn.btn-primary").First().Text() != "ユーザ追加" {
					return fatalErrorf("ユーザ追加ボタンが適切に表示されていません。")
				}
				return nil
			}),
		})
		if err != nil {
			return err
		}

	}
	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil
}

func CheckImage(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
		Description: "存在するユーザでログインできること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               fmt.Sprintf("/static/%s", user.Name+".png"),
		ExpectedStatusCode: 200,
		Description:        "アイコンが表示されること",
	})
	if err != nil {
		return err
	}

	fileName := RandomAlphabetString(20) + ".png"
	title := RandomAlphabetString(16)
	text := RandomAlphabetString(256)
	body, ctype, err := genPostImageBody(fileName, title, text)
	err = checker.Play(ctx, &CheckAction{
		//DisableSlowChecking: true,
		Method:      "POST",
		Path:        "/edudaily/new/",
		ContentType: ctype,
		PostBody:    body,
		CheckFunc:   checkRedirectStatusCode,
		Description: "正常に画像が登録できること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               fmt.Sprintf("/static/%s", fileName),
		ExpectedStatusCode: 200,
		Description:        "登録された画像が表示されること",
		CheckFunc: func(res *http.Response, body *bytes.Buffer) error {
			hasher := md5.New()
			_, err := io.Copy(hasher, body)
			if err != nil {
				return fatalErrorf("レスポンスボディの取得に失敗 %v", err)
			}
			return nil
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil
}

func LoadPostOperation(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/login",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログインできること",
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
	})
	if err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())
	isUpload := rand.Intn(3)
	title := RandomAlphabetString(16)
	text := RandomAlphabetString(256)
	if isUpload == 0 {
		fileName := RandomAlphabetString(20) + ".png"
		body, ctype, err := genPostImageBody(fileName, title, text)
		err = checker.Play(ctx, &CheckAction{
			//DisableSlowChecking: true,
			Method:      "POST",
			Path:        "/edudaily/new/",
			ContentType: ctype,
			PostBody:    body,
			CheckFunc:   checkRedirectStatusCode,
			Description: "新規投稿ができること",
		})
		if err != nil {
			return err
		}
	} else {
		err = checker.Play(ctx, &CheckAction{
			Method:      "POST",
			Path:        "/edudaily/new/",
			CheckFunc:   checkRedirectStatusCode,
			Description: "新規投稿",
			PostData: map[string]string{
				"title": title,
				"text":  text,
			},
		})
		if err != nil {
			return err
		}
	}

	textURL := ""
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/",
		ExpectedStatusCode: 200,
		Description:        "個人ユーザ情報が表示できること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > div .jumbotron > h2").First().Text() != title {
				return fatalErrorf("件名が正常に登録されてません。")
			}
			textURL, _ = doc.Find("body > div .jumbotron > p.lead > a").First().Attr("href")

			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "新規投稿したドキュメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if !strings.Contains(doc.Find("body > div .jumbotron > p").Text(), text) {
				return fatalErrorf("登録した本文が正常に表示されていません")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "POST",
		ExpectedStatusCode: 200,
		Path:               textURL + "/countup",
		Description:        "スターを付与できること",
	})
	if err != nil {
		return err
	}

	newTitle := RandomAlphabetString(24)
	newText := RandomAlphabetString(512)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/update",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿を編集",
		PostData: map[string]string{
			"title": newTitle,
			"text":  newText,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "編集した新規投稿のドキュメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find("body > div .jumbotron > h2").First().Text() != newTitle {
				return fatalErrorf("件名が正常に登録されてません。")
			}

			if !strings.Contains(doc.Find("body > div .jumbotron > p").Text(), newText) {
				return fatalErrorf("編集した本文が正常に表示されていません")
			}
			return nil
		}),
	})
	if err != nil {
		return err
	}

	comText1 := RandomAlphabetString(10)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿へのコメントを追加1",
		PostData: map[string]string{
			"text": comText1,
		},
	})
	if err != nil {
		return err
	}

	comText2 := RandomAlphabetString(20)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿へのコメントを追加2",
		PostData: map[string]string{
			"text": comText2,
		},
	})
	if err != nil {
		return err
	}

	comText3 := RandomAlphabetString(30)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        textURL + "/new_com/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "新規投稿へのコメントを追加3",
		PostData: map[string]string{
			"text": comText3,
		},
	})
	if err != nil {
		return err
	}

	comIDList := []string{}
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               textURL,
		ExpectedStatusCode: 200,
		Description:        "新規投稿したコメントが表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			comment := doc.Find("body > div.container > div.card > div.card-body > p").Text()
			if !strings.Contains(comment, comText1) {
				return fatalErrorf("コメント1 が本文が正常に表示されていません")
			}
			if !strings.Contains(comment, comText2) {
				return fatalErrorf("コメント2 が本文が正常に表示されていません")
			}
			if !strings.Contains(comment, comText3) {
				return fatalErrorf("コメント3 が本文が正常に表示されていません")
			}
			doc.Find("body > div.container > div.card > div.card-body > button").Each(func(_ int, s *goquery.Selection) {
				id, _ := s.Attr("id")
				comIDList = append(comIDList, id)
			})
			return nil
		}),
	})
	if err != nil {
		return err
	}

	for _, comID := range comIDList {
		err = checker.Play(ctx, &CheckAction{
			Method:             "POST",
			ExpectedStatusCode: 200,
			Path:               fmt.Sprintf("/edudaily/%s/countup_com", comID),
			Description:        "コメントに対してスターを付与できること",
		})
	}
	if err != nil {
		return err
	}

	newComTextList := []string{}
	for _, comID := range comIDList {
		newComText := RandomAlphabetString(40)
		newComTextList = append(newComTextList, newComText)
		err = checker.Play(ctx, &CheckAction{
			Method:      "POST",
			Path:        fmt.Sprintf("/edudaily/%s/update_com", comID),
			CheckFunc:   checkRedirectStatusCode,
			Description: "コメントを編集",
			PostData: map[string]string{
				"text": newComText,
			},
		})
	}
	if err != nil {
		return err
	}

	for _, newCom := range newComTextList {
		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               textURL,
			ExpectedStatusCode: 200,
			Description:        "編集したコメントが表示されること",
			CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
				comment := doc.Find("body > div.container > div.card > div.card-body > p").Text()
				if !strings.Contains(comment, newCom) {
					return fatalErrorf("編集したコメントが本文が正常に表示されていません")
				}
				return nil
			}),
		})
	}
	if err != nil {
		return err
	}

	delComID := 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1; i++ {
		delComID = rand.Intn(len(comIDList))
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        fmt.Sprintf("/edudaily/%s/delete_com", comIDList[delComID]),
		CheckFunc:   checkRedirectStatusCode,
		Description: "一部コメントを削除",
	})
	if err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())
	isDel := rand.Intn(2)
	if isDel == 0 {
		err = checker.Play(ctx, &CheckAction{
			Method:      "POST",
			CheckFunc:   checkRedirectStatusCode,
			Path:        textURL + "/delete",
			Description: "新規投稿を削除できること",
		})
		if err != nil {
			return err
		}

		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               textURL,
			ExpectedStatusCode: 404,
			Description:        "新規投稿は 404 になること",
		})
		if err != nil {
			return err
		}
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil
}

func LoadUserOperation(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	_, checker2, push2 := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()
	defer push2()

	err := checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/login",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログインできること",
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/",
		ExpectedStatusCode: 200,
		Description:        "個人ユーザ情報が表示できること",
	})
	if err != nil {
		return err
	}

	user.Password = RandomAlphabetString(16)
	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/user/edit/",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ユーザ情報が更新できること(パスワード)",
		PostData: map[string]string{
			"username": user.Name,
			"password": user.Password,
		},
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "POST",
		Path:        "/login",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログインできること",
		PostData: map[string]string{
			"name":     user.Name,
			"password": user.Password,
		},
	})
	if err != nil {
		return err
	}
	// 管理者ユーザのみ
	if user.IsAdmin == "1" {
		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               "/user/all/",
			ExpectedStatusCode: 200,
			Description:        "全ユーザ情報が表示できること",
		})
		if err != nil {
			return err
		}

		newUser := RandomAlphabetString(14)
		newPass := RandomAlphabetString(14)
		err = checker.Play(ctx, &CheckAction{
			Method: "POST",
			Path:   "/user/add/",
			PostData: map[string]string{
				"username": newUser,
				"password": newPass,
			},
			Description: "新規ユーザが作成できること",
		})
		if err != nil {
			return err
		}

		err = checker2.Play(ctx, &CheckAction{
			Method:      "POST",
			Path:        "/login",
			CheckFunc:   checkRedirectStatusCode,
			Description: "新規ユーザでログインできること",
			PostData: map[string]string{
				"name":     newUser,
				"password": newPass,
			},
		})
		if err != nil {
			return err
		}

		err = checker.Play(ctx, &CheckAction{
			Method:      "POST",
			Path:        fmt.Sprintf("/user/admin/del/%s", newUser),
			CheckFunc:   checkRedirectStatusCode,
			Description: "追加ユーザを削除できること",
		})
		if err != nil {
			return err
		}

		err = checker2.Play(ctx, &CheckAction{
			Method:             "POST",
			Path:               "/login",
			ExpectedStatusCode: 403,
			Description:        "追加ユーザでログインできないこと",
			PostData: map[string]string{
				"name":     newUser,
				"password": newPass,
			},
		})
		if err != nil {
			return err
		}

		err = checker.Play(ctx, &CheckAction{
			Method:      "POST",
			Path:        fmt.Sprintf("/user/admin/back/%s", newUser),
			CheckFunc:   checkRedirectStatusCode,
			Description: "削除したユーザを戻すことができる",
		})
		if err != nil {
			return err
		}

		err = checker2.Play(ctx, &CheckAction{
			Method:      "POST",
			Path:        "/login",
			CheckFunc:   checkRedirectStatusCode,
			Description: "戻したユーザでログインできること",
			PostData: map[string]string{
				"name":     newUser,
				"password": newPass,
			},
		})
		if err != nil {
			return err
		}

		err = checker2.Play(ctx, &CheckAction{
			Method:      "GET",
			Path:        "/logout",
			CheckFunc:   checkRedirectStatusCode,
			Description: "ログアウトできること",
		})
		if err != nil {
			return err
		}

	}

	err = checker.Play(ctx, &CheckAction{
		Method:      "GET",
		Path:        "/logout",
		CheckFunc:   checkRedirectStatusCode,
		Description: "ログアウトできること",
	})
	if err != nil {
		return err
	}

	return nil
}

func LoadReadOperation(ctx context.Context, state *State) error {
	user, checker, push := state.PopRandomUser()
	if user == nil {
		return nil
	}
	defer push()

	err := checker.Play(ctx, &CheckAction{
		Method:    "POST",
		Path:      "/login",
		CheckFunc: checkRedirectStatusCode,
		PostData: map[string]string{
			"name":     "suzuki",
			"password": "suzuki201808",
		},
		Description: "管理者ログインできること",
	})
	if err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())
	page := rand.Perm(8)[0]
	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               fmt.Sprintf("/?page=%d", page),
		ExpectedStatusCode: 200,
		Description:        "記事一覧が表示されること",
		CheckFunc: checkHTML(func(res *http.Response, doc *goquery.Document) error {
			if doc.Find(".jumbotron").Size() != 20 {
				return fatalErrorf("記事総数 20 ありません。")
			}

			return nil
		}),
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/edudaily/new/",
		ExpectedStatusCode: 200,
		Description:        "新規投稿画面が表示されること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/",
		ExpectedStatusCode: 200,
		Description:        "ユーザ画面が表示されること",
	})
	if err != nil {
		return err
	}

	err = checker.Play(ctx, &CheckAction{
		Method:             "GET",
		Path:               "/user/edit/",
		ExpectedStatusCode: 200,
		Description:        "ユーザ編集画面が表示されること",
	})
	if err != nil {
		return err
	}

	for i := 0; i < 20; i++ {
		rand.Seed(time.Now().UnixNano())
		id := rand.Perm(12000)[0]
		id = id + 1
		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               fmt.Sprintf("/edudaily/%d", id),
			ExpectedStatusCode: 200,
			Description:        "記事詳細画面が表示されること",
		})
		if err != nil {
			return err
		}
	}

	for i := 0; i < 5; i++ {
		rand.Seed(time.Now().UnixNano())
		page := rand.Perm(600)[0]
		err = checker.Play(ctx, &CheckAction{
			Method:             "GET",
			Path:               fmt.Sprintf("/?page=%d", page),
			ExpectedStatusCode: 200,
			Description:        "記事一覧が表示されること",
		})
		if err != nil {
			return err
		}

	}

	return nil
}
