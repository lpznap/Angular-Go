import {
  AfterViewInit,
  Component,
  ElementRef,
  forwardRef,
  inject,
  SecurityContext,
  ViewChild,
} from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { DomSanitizer } from '@angular/platform-browser';

@Component({
  selector: 'app-rich-text',
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => RichText), multi: true }],
  template: `<div class="rich-toolbar" role="toolbar" aria-label="Text formatting">
      <button
        type="button"
        (mousedown)="$event.preventDefault()"
        (click)="format('strong')"
        aria-label="Bold"
      >
        <strong>B</strong>
      </button>
      <button
        type="button"
        (mousedown)="$event.preventDefault()"
        (click)="format('em')"
        aria-label="Italic"
      >
        <em>I</em>
      </button>
      <button type="button" (mousedown)="$event.preventDefault()" (click)="format('h2')">
        Heading
      </button>
      <button type="button" (mousedown)="$event.preventDefault()" (click)="format('ul')">
        • List
      </button>
      <button type="button" (mousedown)="$event.preventDefault()" (click)="format('blockquote')">
        Quote
      </button>
    </div>
    <div
      #editor
      class="rich-editor"
      contenteditable="true"
      role="textbox"
      aria-label="THE DETAILS"
      aria-multiline="true"
      data-placeholder="What did you work on? Add context, decisions, or a small win."
      (input)="changed()"
      (blur)="touched()"
      (paste)="paste($event)"
      (drop)="$event.preventDefault()"
    ></div>`,
})
export class RichText implements ControlValueAccessor, AfterViewInit {
  @ViewChild('editor') editor!: ElementRef<HTMLDivElement>;
  private sanitizer = inject(DomSanitizer);
  private pending = '';
  private onChange: (value: string) => void = () => {};
  touched: () => void = () => {};
  ngAfterViewInit() {
    this.writeValue(this.pending);
  }
  writeValue(value: string | null) {
    this.pending = value || '';
    if (this.editor)
      this.editor.nativeElement.innerHTML =
        this.sanitizer.sanitize(SecurityContext.HTML, this.pending) || '';
  }
  registerOnChange(fn: (value: string) => void) {
    this.onChange = fn;
  }
  registerOnTouched(fn: () => void) {
    this.touched = fn;
  }
  setDisabledState(disabled: boolean) {
    if (this.editor) this.editor.nativeElement.contentEditable = String(!disabled);
  }
  changed() {
    this.onChange(this.editor.nativeElement.innerHTML);
  }
  private range(): Range {
    const element = this.editor.nativeElement,
      selection = window.getSelection();
    if (selection?.rangeCount && element.contains(selection.getRangeAt(0).commonAncestorContainer))
      return selection.getRangeAt(0);
    const range = document.createRange();
    range.selectNodeContents(element);
    range.collapse(false);
    return range;
  }
  format(tag: string) {
    const range = this.range(),
      wrapper = document.createElement(tag);
    const content = range.extractContents();
    if (!content.textContent) content.append(document.createTextNode('Text'));
    if (tag === 'ul') {
      const item = document.createElement('li');
      item.append(content);
      wrapper.append(item);
    } else wrapper.append(content);
    range.insertNode(wrapper);
    range.selectNodeContents(wrapper);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
    this.editor.nativeElement.focus();
    this.changed();
  }
  paste(event: ClipboardEvent) {
    event.preventDefault();
    const text = event.clipboardData?.getData('text/plain') || '';
    const range = this.range();
    range.deleteContents();
    const node = document.createTextNode(text);
    range.insertNode(node);
    range.setStartAfter(node);
    range.collapse(true);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
    this.changed();
  }
}
