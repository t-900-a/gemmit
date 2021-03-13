const { Readability } = require('@mozilla/readability');
const jsdom = require('jsdom');
const JSDOM = jsdom.JSDOM;
const fetch = require('node-fetch');
const sanitizer = require('sanitize-html');
const collapse = require('collapse-white-space');
const Entities = require('html-entities').AllHtmlEntities;
const entities = new Entities();

jsdom.defaultDocumentFeatures = {
  QuerySelector: true
};

const sanitize = html => entities.decode(sanitizer(collapse(html), {
  allowedTags: [],
  allowedAttributes: {},
})).trim();
const sanitizePre = html => entities.decode(sanitizer(html, {
  allowedTags: [],
  allowedAttributes: {},
})).trim();

const convert = (dom, title) => {
  let output = "";
  const elements = [...dom.querySelectorAll(`
    .page h1, .page h2, .page h3, .page h4, .page h5,
    .page p,
    .page img,
    .page blockquote,
    .page pre,
    .page a,
    .page ul, .page ol
  `)];
  if (elements.length === 0) {
    return "Unable to process this URL\n";
  }
  if (elements[0].tagName.toLowerCase() !== "h1") {
    output += `# ${title}\n\n`;
  }

  const rewriteLink = href =>
    "?" + encodeURIComponent(href);

  const emitLink = link => {
    let desc = sanitize(link.innerHTML);
    if (desc === "") {
      const img = link.querySelector("img");
      if (img) {
        desc = `${img.alt ? " " + img.alt : ""}\n\n`;
      }
    }
    if (link.protocol === "http:" || link.protocol === "https:") {
      output += `=> ${rewriteLink(link.href)} ${desc}\n`;
    } else {
      output += `=> ${link.href} ${desc}\n`;
    }
  };

  let visited = [];
  for (let i = 0; i < elements.length; i++) {
    let el = elements[i];
    if (visited.filter(v => v.contains(el)).length !== 0) {
      continue;
    }
    visited.push(el);

    let links = [];
    switch (el.tagName.toLowerCase()) {
    case 'p':
        output += sanitize(el.innerHTML) + "\n\n";
        links = [...el.querySelectorAll("a")];
        links.map(emitLink);
        if (links.length !== 0) {
          output += "\n";
        }
        break;
    case 'a':
        emitLink(el);
        break;
    case 'ul':
    case 'ol':
        links = [...el.querySelectorAll("a")];
        [...el.children].map(item => {
          output += `* ${sanitize(item.innerHTML)}\n`;
        });
        output += "\n";

        if (links.length !== 0) {
          output += "\n";
          links.map(emitLink);
          output += "\n";
        }
        break;
    case 'h1':
        output += `# ${sanitize(el.innerHTML)}\n\n`
        break;
    case 'h2':
        output += `## ${sanitize(el.innerHTML)}\n\n`
        break;
    case 'h3':
    case 'h4':
    case 'h5':
        output += `### ${sanitize(el.innerHTML)}\n\n`
        break;
    case 'img':
        output += `=> ${el.src} ${el.alt ? el.alt : "(image)"}\n\n`
        break;
    case 'pre':
        output += "```\n";
        output += sanitizePre(el.innerHTML) + "\n\n";
        output += "```\n";
        break;
    case 'blockquote':
        output += `> ${sanitize(el.innerHTML)}\n\n`;
        break;
    }
  }
  return output.replace(/\n\n\n+/g, "\n\n").trim();
};

const args = process.argv.slice(2);
fetch(args[0])
  .then(res => {
    if (!res.ok) {
      throw "Received non-200 response from server: " + res.status;
    }
    let ct = res.headers.get('content-type');
    if (ct.indexOf(';') !== -1) {
      ct = ct.slice(0, ct.indexOf(';'));
    }
    if (ct !== "text/html") {
      throw "Received non-HTML response " + ct;
    }
    return res.text();
  })
  .then(body => {
    const doc = new JSDOM(body, {
      url: args[0],
    });
    const reader = new Readability(doc.window.document);
    const article = reader.parse();
    const readable = new JSDOM(article.content, {url: args[0]});
    console.log(convert(readable.window.document, article.title));
  })
  .catch(err => console.log("An error occured while fetching this page: " + err));
